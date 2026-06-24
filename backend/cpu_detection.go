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

	// Also query the manufacturer string so we can differentiate Intel vs AMD
	// — they need different bit patterns for the same "half cores" goal.
	psScript := `
$cpu = Get-WmiObject Win32_Processor | Select-Object -First 1
$physical = [int]$cpu.NumberOfCores
$logical = [int]$cpu.NumberOfLogicalProcessors
$mfr = $cpu.Manufacturer
Write-Output "$physical $logical $mfr"
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.Output()
	if err != nil {
		logs = append(logs, fmt.Sprintf("Warning: PowerShell query failed: %v — using fallback.", err))
		return fallbackMask, logs
	}

	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) < 2 {
		logs = append(logs, fmt.Sprintf("Warning: unexpected PowerShell output %q — using fallback.", strings.TrimSpace(string(out))))
		return fallbackMask, logs
	}

	physicalCores, err1 := strconv.Atoi(parts[0])
	logicalCores, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || physicalCores == 0 {
		logs = append(logs, "Warning: failed to parse core counts — using fallback.")
		return fallbackMask, logs
	}

	// Manufacturer string — "GenuineIntel" or "AuthenticAMD"
	manufacturer := ""
	if len(parts) >= 3 {
		manufacturer = strings.ToLower(strings.Join(parts[2:], " "))
	}
	isIntel := strings.Contains(manufacturer, "intel")
	isAMD := strings.Contains(manufacturer, "amd")

	logs = append(logs, fmt.Sprintf("Physical cores: %d, Logical processors: %d, Manufacturer: %s", physicalCores, logicalCores, manufacturer))

	hasHT := logicalCores > physicalCores

	if physicalCores > 64 {
		physicalCores = 64
	}

	coresToTarget := physicalCores / 2
	if coresToTarget == 0 {
		coresToTarget = 1
	}

	var finalMask uint64

	switch {
	case isIntel && hasHT:
		// Intel HT: every other bit — one primary thread per physical core.
		// i7-13700KF: 16 physical / 2 = 8 cores → 5555 5555 & keep lower 16 bits → 5555
		logs = append(logs, fmt.Sprintf("Intel HT architecture. Targeting %d P-cores (every other logical processor).", coresToTarget))
		for i := 0; i < coresToTarget; i++ {
			finalMask |= 1 << uint(i*2)
		}

	case isAMD:
		// AMD SMT: consecutive bits — half the physical cores starting from 0.
		// 7900X: 12 physical / 2 = 6 cores → 111111b = 0x3F... but community uses 555
		// 555 hex = 0101 0101 0101 = every other bit on 12 logical processors
		// So AMD also uses every-other-bit but on logical (not physical) count.
		// 12 logical / 2 = 6 bits set at positions 0,2,4,6,8,10 → 555
		logs = append(logs, fmt.Sprintf("AMD SMT architecture. Targeting %d logical processors (every other).", logicalCores/2))
		logicalToTarget := logicalCores / 2
		for i := 0; i < logicalToTarget; i++ {
			finalMask |= 1 << uint(i*2)
		}

	default:
		// Unknown/no HT: consecutive bits up to half physical cores
		logs = append(logs, fmt.Sprintf("Standard architecture. Targeting %d cores consecutively.", coresToTarget))
		finalMask = (uint64(1) << uint(coresToTarget)) - 1
	}

	if finalMask == 0 {
		logs = append(logs, "Warning: mask resolved to zero — deploying fallback.")
		return fallbackMask, logs
	}

	hexMask := fmt.Sprintf("%X", finalMask)
	logs = append(logs, fmt.Sprintf("Optimal affinity mask resolved: 0x%s", hexMask))
	return hexMask, logs
}