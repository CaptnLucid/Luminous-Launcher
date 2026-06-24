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

	cmd := exec.Command(exePath, args...)
	cmd.Dir = filepath.Dir(exePath)

	if err := cmd.Start(); err != nil {
		return err
	}

	mask, err := strconv.ParseUint(hexMask, 16, 64)
	if err != nil || mask == 0 {
		return nil
	}

	// Apply affinity in a goroutine so we don't block the UI, but wait 500ms
	// first to let the launcher finish initializing — matches the synchronous
	// behaviour of the Windows "Start /affinity" command.
	go func() {
		time.Sleep(500 * time.Millisecond)
		applyWin32AffinityMask(cmd.Process.Pid, uintptr(mask))
	}()

	return nil
}

func applyWin32AffinityMask(pid int, mask uintptr) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	openProcess := kernel32.NewProc("OpenProcess")
	setAffinity := kernel32.NewProc("SetProcessAffinityMask")

	handle, _, _ := openProcess.Call(0x0200, 0, uintptr(pid))
	if handle != 0 {
		defer syscall.CloseHandle(syscall.Handle(handle))
		_, _, _ = setAffinity.Call(handle, mask)
	}
}