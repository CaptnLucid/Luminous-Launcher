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
	fallbackMask := fmt.Sprintf("%04X", (uint64(1)<<totalCores)-1)
	logs = append(logs, fmt.Sprintf("Logical processors detected: %d (fallback mask: %s)", totalCores, fallbackMask))

	psScript := `
$cpu = Get-WmiObject Win32_Processor | Select-Object -First 1
$physical = [int]$cpu.NumberOfCores
$logical = [int]$cpu.NumberOfLogicalProcessors
Write-Output "$physical $logical"
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)

	// Hide the PowerShell console window — without this a terminal flashes briefly on screen.
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
		logs = append(logs, "Hyperthreading / hybrid architecture detected. Targeting one thread per physical core.")
	} else {
		logs = append(logs, "Uniform core architecture detected (no HT). Targeting all logical processors.")
	}

	if physicalCores > 64 {
		physicalCores = 64
	}
	if logicalCores > 64 {
		logicalCores = 64
	}

	var finalMask uint64
	if hasHT {
		for i := 0; i < physicalCores; i++ {
			finalMask |= 1 << uint(i*2)
		}
	} else {
		finalMask = (uint64(1) << uint(physicalCores)) - 1
	}

	if finalMask == 0 {
		logs = append(logs, "Warning: mask resolved to zero — deploying fallback.")
		return fallbackMask, logs
	}

	finalMask = finalMask & 0xFFFF

	// %04X = uppercase hex, minimum 4 characters, zero-padded.
	// Keeps the field consistent (e.g. 5555 not 55555555).
	hexMask := fmt.Sprintf("%04X", finalMask)
	logs = append(logs, fmt.Sprintf("Optimal affinity mask resolved: 0x%s", hexMask))
	return hexMask, logs
}