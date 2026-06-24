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

	totalCores := runtime.NumCPU()
	if totalCores > 64 {
		totalCores = 64
	}
	fallbackMask := fmt.Sprintf("%X", (uint64(1)<<totalCores)-1)
	logs = append(logs, fmt.Sprintf("Logical processors detected: %d (fallback mask: %s)", totalCores, fallbackMask))

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

	hasHT := logicalCores > physicalCores
	if hasHT {
		logs = append(logs, "Hyperthreading / SMT detected. Targeting one thread per physical core.")
	} else {
		logs = append(logs, "Uniform core architecture detected (no SMT). Targeting all logical processors.")
	}

	if physicalCores > 64 {
		physicalCores = 64
	}

	// Target half the physical cores — one thread each.
	// This matches the BDO community recommended affinity values:
	//   i7-13700KF (8 P-cores) → 4 cores → 5555
	//   AMD 7900X  (12 cores)  → 6 cores → 555
	// Using half avoids thermal/scheduling contention while keeping
	// the game on the fastest cores.
	coresToTarget := physicalCores / 2
	if coresToTarget == 0 {
		coresToTarget = 1
	}

	var finalMask uint64
	if hasHT {
		// Every other bit — one primary thread per physical core
		for i := 0; i < coresToTarget; i++ {
			finalMask |= 1 << uint(i*2)
		}
	} else {
		// Consecutive bits — one thread per core, no HT sibling to skip
		finalMask = (uint64(1) << uint(coresToTarget)) - 1
	}

	if finalMask == 0 {
		logs = append(logs, "Warning: mask resolved to zero — deploying fallback.")
		return fallbackMask, logs
	}

	// No zero-padding — let the mask be as wide as it naturally needs to be.
	// 13700KF → 5555 (4 P-cores × HT = 8 bits = 4 hex chars)
	// 7900X   → 555  (6 cores / 2 = 3 hex chars)
	hexMask := fmt.Sprintf("%X", finalMask)
	logs = append(logs, fmt.Sprintf("Optimal affinity mask resolved: 0x%s (%d cores targeted)", hexMask, coresToTarget))
	return hexMask, logs
}