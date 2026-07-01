package backend

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func InjectNipProfile(baseDir, nipPath string) error {
	inspectorExe := filepath.Join(baseDir, "nvidiaProfileInspector", "nvidiaProfileInspector.exe")
	if _, err := os.Stat(inspectorExe); os.IsNotExist(err) {
		return errors.New("nvidiaProfileInspector.exe is missing inside subdirectory")
	}

	cmd := exec.Command(inspectorExe, nipPath, "-silent")
	return cmd.Run()
}

// knownGameProcessNames covers the executables that can end up running the actual
// game session once the launcher chain hands off. The Black Desert launcher
// frequently exits and is replaced by a different process (the real client),
// so the PID handed back by cmd.Start() is not, by itself, a reliable target.
var knownGameProcessNames = []string{
	"blackdesert64.exe",
	"blackdesert32.exe",
	"blackdesertlauncher.exe",
	"bdolauncher.exe",
}

// SpawnGameWithAffinity launches the target executable and then keeps watching
// the resulting process tree (plus any known game executables that appear) for
// a window of time, applying the requested affinity mask to every match. This
// is what actually makes the affinity "stick" - a single apply-once attempt
// against the launcher's original PID is not enough, since that process is
// commonly short-lived and the real game process starts moments later.
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

	launchedPid := uint32(cmd.Process.Pid)

	go watchAndApplyAffinity(launchedPid, uintptr(mask))

	return nil
}

// watchAndApplyAffinity polls the process tree for up to a minute after launch,
// applying the affinity mask to the original process, any descendants it spawns,
// and any separately-started process matching a known game executable name -
// covering the case where the launcher replaces itself rather than forking.
func watchAndApplyAffinity(rootPid uint32, mask uintptr) {
	applied := make(map[uint32]bool)
	deadline := time.Now().Add(60 * time.Second)

	// Give the freshly spawned process a moment to finish initializing.
	time.Sleep(1000 * time.Millisecond)

	consecutiveStaleScans := 0

	for time.Now().Before(deadline) {
		candidates := collectCandidatePids(rootPid)

		newlyApplied := false
		for _, pid := range candidates {
			if applied[pid] {
				continue
			}
			if err := applyWin32AffinityMask(pid, mask); err == nil {
				applied[pid] = true
				newlyApplied = true
			}
		}

		if newlyApplied {
			consecutiveStaleScans = 0
		} else if len(applied) > 0 {
			consecutiveStaleScans++
		}

		// Once we've successfully pinned at least one process and haven't found
		// any new descendants for a few scans in a row, the hand-off has settled.
		if len(applied) > 0 && consecutiveStaleScans >= 3 {
			return
		}

		time.Sleep(1500 * time.Millisecond)
	}
}

// collectCandidatePids returns the root pid (if still alive), every descendant
// of it, and any independently running process matching a known game executable
// name.
func collectCandidatePids(rootPid uint32) []uint32 {
	entries, err := snapshotProcesses()
	if err != nil {
		return []uint32{rootPid}
	}

	var result []uint32
	seen := make(map[uint32]bool)

	if pidExists(entries, rootPid) {
		seen[rootPid] = true
		result = append(result, rootPid)
	}

	var addDescendants func(parent uint32)
	addDescendants = func(parent uint32) {
		for _, e := range entries {
			if e.ParentPid == parent && !seen[e.Pid] {
				seen[e.Pid] = true
				result = append(result, e.Pid)
				addDescendants(e.Pid)
			}
		}
	}
	addDescendants(rootPid)

	for _, e := range entries {
		nameLower := strings.ToLower(e.Name)
		for _, known := range knownGameProcessNames {
			if nameLower == known && !seen[e.Pid] {
				seen[e.Pid] = true
				result = append(result, e.Pid)
			}
		}
	}

	return result
}

func pidExists(entries []procEntry, pid uint32) bool {
	for _, e := range entries {
		if e.Pid == pid {
			return true
		}
	}
	return false
}

type procEntry struct {
	Pid       uint32
	ParentPid uint32
	Name      string
}

const th32csSnapProcess = 0x00000002

type processEntry32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

// snapshotProcesses walks the current process table via the classic
// Toolhelp32 snapshot APIs, returning pid/parent-pid/name triples we can use
// to find descendants of the launched process.
func snapshotProcesses() ([]procEntry, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	createSnapshot := kernel32.NewProc("CreateToolhelp32Snapshot")
	process32First := kernel32.NewProc("Process32FirstW")
	process32Next := kernel32.NewProc("Process32NextW")

	handle, _, _ := createSnapshot.Call(uintptr(th32csSnapProcess), 0)
	if handle == uintptr(syscall.InvalidHandle) || handle == 0 {
		return nil, errors.New("failed to create process snapshot")
	}
	snapHandle := syscall.Handle(handle)
	defer syscall.CloseHandle(snapHandle)

	var entries []procEntry
	var pe processEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))

	ret, _, _ := process32First.Call(uintptr(snapHandle), uintptr(unsafe.Pointer(&pe)))
	if ret == 0 {
		return nil, errors.New("Process32First failed")
	}

	for {
		entries = append(entries, procEntry{
			Pid:       pe.ProcessID,
			ParentPid: pe.ParentProcessID,
			Name:      syscall.UTF16ToString(pe.ExeFile[:]),
		})

		ret, _, _ := process32Next.Call(uintptr(snapHandle), uintptr(unsafe.Pointer(&pe)))
		if ret == 0 {
			break
		}
		// Reset Size each iteration since some implementations are picky about it.
		pe.Size = uint32(unsafe.Sizeof(pe))
	}

	return entries, nil
}

func applyWin32AffinityMask(pid uint32, mask uintptr) error {
	const (
		PROCESS_SET_INFORMATION   = 0x0200
		PROCESS_QUERY_INFORMATION = 0x0400
	)

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	openProcess := kernel32.NewProc("OpenProcess")
	setProcessAffinityMask := kernel32.NewProc("SetProcessAffinityMask")
	getLastError := kernel32.NewProc("GetLastError")

	// Open process with BOTH required flags.
	// Without PROCESS_QUERY_INFORMATION, some Windows versions reject the call.
	handle, _, _ := openProcess.Call(
		uintptr(PROCESS_SET_INFORMATION|PROCESS_QUERY_INFORMATION),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		errCode, _, _ := getLastError.Call()
		return errors.New("failed to open process (error code: " + strconv.FormatUint(uint64(errCode), 10) + " - may need admin rights or process may have exited)")
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	// SetProcessAffinityMask returns non-zero on success, zero on failure.
	ret, _, winErr := setProcessAffinityMask.Call(handle, mask)
	if ret == 0 {
		errCode, _, _ := getLastError.Call()
		return errors.New("SetProcessAffinityMask failed: " + winErr.Error() + " (error code: " + strconv.FormatUint(uint64(errCode), 10) + ")")
	}

	return nil
}