// backend/cpu_database.go
//
// Hardcoded reference table of known-good CPU affinity masks for AMD Ryzen
// processors running Black Desert Online, sourced from the community guide
// "CPU Performance - Ryzen CPU Affinities" (credit: ACanadianDude,
// https://linktr.ee/ACanadianDude). This lets the launcher recommend a
// known-good mask by CPU model name, separate from the generic topology-based
// heuristic in cpu_detection.go.
package backend

import (
	"strings"
	"syscall"
	"unsafe"
)

// CPUAffinityEntry describes a single recommended affinity profile for one or
// more CPU models that share an identical core/CCX layout.
type CPUAffinityEntry struct {
	Models       []string `json:"models"`       // lowercase substrings matched against the detected CPU name
	Label        string   `json:"label"`        // human readable model grouping
	Architecture string   `json:"architecture"` // e.g. "Zen 2"
	Cores        string   `json:"cores"`        // e.g. "8C / 16T"
	Enable       string   `json:"enable"`       // recommended cores to enable
	AffinityMask string   `json:"affinityMask"` // recommended hex mask
	Note         string   `json:"note"`         // any caveats from the source guide
}

// ryzenAffinityDatabase is the hardcoded table transcribed from the
// "CPU Performance - Ryzen CPU Affinities" reference document.
var ryzenAffinityDatabase = []CPUAffinityEntry{
	{
		Models:       []string{"ryzen 3 1200", "ryzen 3 1300x", "ryzen 3 2200g", "ryzen 3 2200ge", "ryzen 3 3200g", "ryzen 3 3200ge"},
		Label:        "Ryzen 3 1200 / 1300X / 2200G(E) / 3200G(E)",
		Architecture: "Zen / Zen+",
		Cores:        "4C / 4T",
		Enable:       "2, 3",
		AffinityMask: "C",
		Note:         "May have too few cores for this optimization to provide noticeable benefits.",
	},
	{
		Models:       []string{"ryzen 5 1400", "ryzen 5 1500x", "ryzen 5 2400g", "ryzen 5 2400ge", "ryzen 5 2500x", "ryzen 5 3400g", "ryzen 5 3400ge"},
		Label:        "Ryzen 5 1400 / 1500X / 2400G(E) / 2500X / 3400G(E)",
		Architecture: "Zen / Zen+",
		Cores:        "4C / 8T",
		Enable:       "4, 6",
		AffinityMask: "50",
		Note:         "If performance is worse, try using cores 0, 2, 4, 6 instead.",
	},
	{
		Models:       []string{"ryzen 5 1600", "ryzen 5 1600x", "ryzen 5 2600", "ryzen 5 2600x"},
		Label:        "Ryzen 5 1600(X) / 2600(X)",
		Architecture: "Zen / Zen+",
		Cores:        "6C / 12T",
		Enable:       "6, 8, 10",
		AffinityMask: "540",
		Note:         "Alternative: enable 6, 7, 8, 9, 10, 11 if performance decreases.",
	},
	{
		Models:       []string{"ryzen 7 1700", "ryzen 7 1700x", "ryzen 7 1800", "ryzen 7 1800x", "ryzen 7 2700", "ryzen 7 2700x"},
		Label:        "Ryzen 7 1700(X) / 1800(X) / 2700(X)",
		Architecture: "Zen / Zen+",
		Cores:        "8C / 16T",
		Enable:       "8, 10, 12, 14",
		AffinityMask: "5500",
	},
	{
		Models:       []string{"ryzen 5 3500", "ryzen 5 3500x"},
		Label:        "Ryzen 5 3500(X)",
		Architecture: "Zen 2",
		Cores:        "6C / 6T",
		Enable:       "(no changes required)",
		AffinityMask: "0",
		Note:         "No changes required.",
	},
	{
		Models:       []string{"ryzen 5 3600", "ryzen 5 3600x", "ryzen 5 5600", "ryzen 5 5600x", "ryzen 5 7600", "ryzen 5 7600x", "ryzen 5 9600", "ryzen 5 9600x"},
		Label:        "Ryzen 5 3600(X) / 5600(X) / 7600(X) / 9600(X)",
		Cores:        "6C / 12T",
		Enable:       "0, 2, 4, 6, 8, 10",
		AffinityMask: "555",
		Note:         "Limit the game to 6 physical cores.",
	},
	{
		Models:       []string{"ryzen 7 3700", "ryzen 7 3700x", "ryzen 7 3800", "ryzen 7 3800x"},
		Label:        "Ryzen 7 3700(X) / 3800X",
		Architecture: "Zen 2",
		Cores:        "8C / 16T",
		Enable:       "4, 6, 8, 10, 12, 14",
		AffinityMask: "5550",
		Note:         "Limit the game to 6 physical cores.",
	},
	{
		Models: []string{
			"ryzen 7 5800x3d", "ryzen 7 5800x", "ryzen 7 7700x", "ryzen 7 7800x3d",
			"ryzen 7 9700x", "ryzen 7 9800x3d",
		},
		Label:        "Ryzen 7 5800X / 5800X3D / 7700X / 7800X3D / 9700X / 9800X3D",
		Cores:        "8C / 16T",
		Enable:       "2, 4, 6, 8, 10, 12, 14",
		AffinityMask: "5554",
		Note:         "Disable Core 0 to leave headroom for Windows background tasks.",
	},
	{
		Models:       []string{"ryzen 9 3900", "ryzen 9 3900x", "ryzen 9 5900x", "ryzen 9 7900x", "ryzen 9 9900x"},
		Label:        "Ryzen 9 3900(X) / 5900X / 7900X / 9900X",
		Architecture: "Zen 2 or newer",
		Cores:        "12C / 24T",
		Enable:       "12, 14, 16, 18, 20, 22",
		AffinityMask: "555000",
		Note:         "Isolate BDO to a single CCD.",
	},
	{
		Models:       []string{"ryzen 9 7900x3d", "ryzen 9 9900x3d"},
		Label:        "Ryzen 9 7900X3D / 9900X3D",
		Architecture: "Zen 4",
		Cores:        "12C / 24T",
		Enable:       "0, 2, 4, 6, 8, 10",
		AffinityMask: "555",
		Note:         "X3D cache only on CCD 1 - isolate BDO to the CCD containing the X3D cache.",
	},
	{
		Models:       []string{"ryzen 9 3950x", "ryzen 9 5950x", "ryzen 9 7950x"},
		Label:        "Ryzen 9 3950X / 5950X / 7950X",
		Architecture: "Zen 2 or newer",
		Cores:        "16C / 32T",
		Enable:       "16, 18, 20, 22, 24, 26",
		AffinityMask: "5550000",
		Note:         "Isolate BDO to a single chiplet, limited to 6 physical cores.",
	},
	{
		Models:       []string{"ryzen 9 7950x3d", "ryzen 9 9950x3d"},
		Label:        "Ryzen 9 7950X3D / 9950X3D",
		Architecture: "Zen 4",
		Cores:        "16C / 32T",
		Enable:       "0, 2, 4, 6, 8, 10, 12, 14",
		AffinityMask: "5555",
		Note:         "X3D cache only on CCD 1 - isolate BDO to the CCD containing the X3D cache.",
	},
}

// DefaultAffinityMask is used whenever the detected CPU has no known entry.
const DefaultAffinityMask = "FFFF"

// MatchKnownAffinity looks up the best-matching hardcoded entry for the given
// CPU model name string (as returned by GetCPUModelName). The longest matching
// model substring wins, so more specific models (e.g. "ryzen 9 7950x3d") are
// preferred over shorter ones that might also match (e.g. "ryzen 9 7950x").
func MatchKnownAffinity(cpuName string) (CPUAffinityEntry, bool) {
	name := strings.ToLower(cpuName)

	var best CPUAffinityEntry
	bestLen := -1
	found := false

	for _, entry := range ryzenAffinityDatabase {
		for _, model := range entry.Models {
			if strings.Contains(name, model) && len(model) > bestLen {
				best = entry
				bestLen = len(model)
				found = true
			}
		}
	}

	return best, found
}

// GetCPUModelName reads the processor name string directly from the Windows
// registry (HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0). This avoids
// pulling in WMI just to get a model string. Returns "" if it can't be read
// (e.g. running outside Windows, or insufficient permissions).
func GetCPUModelName() string {
	keyPath, err := syscall.UTF16PtrFromString(`HARDWARE\DESCRIPTION\System\CentralProcessor\0`)
	if err != nil {
		return ""
	}

	var key syscall.Handle
	err = syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, keyPath, 0, syscall.KEY_READ, &key)
	if err != nil {
		return ""
	}
	defer syscall.RegCloseKey(key)

	valueName, err := syscall.UTF16PtrFromString("ProcessorNameString")
	if err != nil {
		return ""
	}

	var bufLen uint32 = 512
	buf := make([]uint16, bufLen/2)
	var valType uint32

	err = syscall.RegQueryValueEx(
		key,
		valueName,
		nil,
		&valType,
		(*byte)(unsafe.Pointer(&buf[0])),
		&bufLen,
	)
	if err != nil {
		return ""
	}

	return strings.TrimRight(syscall.UTF16ToString(buf), " \x00")
}

// RecommendedAffinity bundles the detected CPU name together with whichever
// affinity recommendation applies to it - a known database match if found,
// otherwise the safe FFFF default.
type RecommendedAffinity struct {
	CPUName      string `json:"cpuName"`
	Matched      bool   `json:"matched"`
	Label        string `json:"label"`
	AffinityMask string `json:"affinityMask"`
	Enable       string `json:"enable"`
	Note         string `json:"note"`
}

// GetRecommendedAffinity is the entry point used by the frontend's
// "known CPU" detection button: it identifies the installed CPU and returns
// the matching hardcoded recommendation, or falls back to FFFF if the CPU
// isn't in the database (or couldn't be identified at all).
func GetRecommendedAffinity() RecommendedAffinity {
	cpuName := GetCPUModelName()

	entry, matched := MatchKnownAffinity(cpuName)
	if !matched {
		return RecommendedAffinity{
			CPUName:      cpuName,
			Matched:      false,
			AffinityMask: DefaultAffinityMask,
		}
	}

	return RecommendedAffinity{
		CPUName:      cpuName,
		Matched:      true,
		Label:        entry.Label,
		AffinityMask: entry.AffinityMask,
		Enable:       entry.Enable,
		Note:         entry.Note,
	}
}