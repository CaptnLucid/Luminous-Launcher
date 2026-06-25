package backend

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
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

	gameDir := filepath.Dir(exePath)
	gameExe := filepath.Base(exePath)

	var args string
	if isSteam {
		args = " -steam"
	}

	mask, err := strconv.ParseUint(hexMask, 16, 64)
	if err != nil || mask == 0 {
		mask = 0xFFFF
	}

	cmdStr := fmt.Sprintf(`cd /d "%s" && Start /affinity %X %s%s`, gameDir, mask, gameExe, args)
	cmd := exec.Command("cmd", "/C", cmdStr)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	return cmd.Start()
}