package backend

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

func InjectNipProfile(baseDir, nipPath string) error {
	inspectorExe := filepath.Join(baseDir, "nvidiaProfileInspector", "nvidiaProfileInspector.exe")
	if _, err := os.Stat(inspectorExe); os.IsNotExist(err) {
		return errors.New("nvidiaProfileInspector.exe is missing inside subdirectory")
	}

	cmd := exec.Command(inspectorExe, nipPath, "-silent")
	return cmd.Run()
}

func SpawnGameWithAffinity(exePath string, isSteam bool, hexMask string) error {
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return errors.New("target game executable path could not be found")
	}

	var args []string
	if isSteam {
		args = append(args, "-steam")
	}

	mask, err := strconv.ParseUint(hexMask, 16, 64)
	if err != nil || mask == 0 {
		mask = 0xFFFF
	}

	cmd := exec.Command(exePath, args...)
	cmd.Dir = filepath.Dir(exePath)

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		applyWin32AffinityMask(cmd.Process.Pid, uintptr(mask))
	}()

	return nil
}

func applyWin32AffinityMask(pid int, mask uintptr) {
	const PROCESS_SET_INFORMATION = 0x0200
	
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	openProcess := kernel32.NewProc("OpenProcess")
	setProcessAffinityMask := kernel32.NewProc("SetProcessAffinityMask")

	// 1. Open process handle with permissions to modify configuration
	handle, _, _ := openProcess.Call(
		uintptr(PROCESS_SET_INFORMATION),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	// 2. Safely commit the hex allocation directly to the Windows scheduler
	_, _, _ = setProcessAffinityMask.Call(handle, mask)
}