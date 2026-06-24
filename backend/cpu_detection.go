package backend

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// DetectOptimalAffinityMask returns the optimal hex affinity mask and debug log lines.
// Uses PowerShell to query CPU topology safely — no unsafe pointers, no syscall hazards.
func DetectOptimalAffinityMask() (string, []string) {
	var logs []string
	logs = append(logs, "Starting CPU topology detection via PowerShell query...")

	// Runtime fallback in case PowerShell fails
	totalLogical := runtime.NumCPU()
	if totalLogical > 64 {
		totalLogical = 64
	}
	fallbackMask := fmt.Sprintf("%X", (uint64(1)<<uint(totalLogical/2))-1)
	logs = append(logs, fmt.Sprintf("Runtime logical processors: %d (fallback mask: %s)", totalLogical, fallbackMask))

	psScript := `
$cpu = Get-WmiObject Win32_Processor | Select-Object -First 1
$physical = [int]$cpu.NumberOfCores
$logical = [int]$cpu.NumberOfLogicalProcessors
Write-Output "$physical $logical"
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.Output()
	if err != nil {
		logs = append(logs, fmt.Sprintf("Warning: PowerShell query failed: %v — using fallback.", err))
		return fallbackMask, logs
	}

	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		logs = append(logs, fmt.Sprintf("Warning: unexpected PowerShell output %q — using fallback.", strings.TrimSpace(string(out))))
		return fallbackMask, logs
	}

	physicalCores, err1 := strconv.Atoi(parts[0])
	logicalCores, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || physicalCores == 0 {
		logs = append(logs, "Warning: failed to parse core counts — using fallback.")
		return fallbackMask, logs
	}

	logs = append(logs, fmt.Sprintf("Physical cores: %d, Logical processors: %d", physicalCores, logicalCores))

	// How many logical processors per physical core (1 = no SMT, 2 = HT/SMT)
	// Use ceiling division to handle hybrid architectures (e.g., Intel P/E-cores)
	threadsPerCore := (logicalCores + physicalCores - 1) / physicalCores
	hasSMT := threadsPerCore > 1
	if hasSMT {
		logs = append(logs, fmt.Sprintf("SMT/HT detected (%d threads per core).", threadsPerCore))
	} else {
		logs = append(logs, "No SMT detected (1 thread per core).")
	}

	// Target half the physical cores — universally recommended for BDO
	// to avoid thermal contention while keeping the game on fastest cores.
	coresToTarget := physicalCores / 2
	if coresToTarget == 0 {
		coresToTarget = 1
	}
	logs = append(logs, fmt.Sprintf("Targeting %d of %d physical cores.", coresToTarget, physicalCores))

	// Build the mask by targeting the first logical processor of each
	// physical core. With SMT, physical core N owns logical processors
	// N*threadsPerCore and N*threadsPerCore+1, so we take the first one.
	// Without SMT, logical == physical so we just set consecutive bits.
	var finalMask uint64
	for i := 0; i < coresToTarget; i++ {
		logicalIndex := i * threadsPerCore
		finalMask |= 1 << uint(logicalIndex)
	}

	if finalMask == 0 {
		logs = append(logs, "Warning: mask resolved to zero — deploying fallback.")
		return fallbackMask, logs
	}

	hexMask := fmt.Sprintf("%X", finalMask)
	logs = append(logs, fmt.Sprintf("Optimal affinity mask resolved: %s", hexMask))
	return hexMask, logs
}