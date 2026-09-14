package netskope

import (
	"os"
	"path/filepath"
	"runtime"
)

// installPathCandidates returns the directories the Netskope installer uses for
// the STAgent on the running platform, most specific first. An unsupported
// platform returns nil, which reports the client as not installed.
func installPathCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/Library/Application Support/Netskope/STAgent"}
	case "windows":
		return []string{
			`C:\Program Files\Netskope\STAgent`,
			`C:\Program Files (x86)\Netskope\STAgent`,
		}
	case "linux":
		return []string{
			"/opt/netskope/stagent",
			"/opt/Netskope/STAgent",
		}
	}
	return nil
}

// findInstallPath returns the first candidate that holds the nsdiag binary,
// falling back to the first candidate that exists as a directory so that a
// present-but-unusable install is still reported (with an error) rather than
// looking like no install at all.
func findInstallPath(candidates []string, stat func(string) (os.FileInfo, error)) string {
	for _, candidate := range candidates {
		bin := filepath.Join(candidate, nsdiagBinaryName())
		if st, err := stat(bin); err == nil && !st.IsDir() {
			return candidate
		}
	}

	for _, candidate := range candidates {
		if st, err := stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
	}

	return ""
}

func nsdiagBinaryName() string {
	if runtime.GOOS == "windows" {
		return "nsdiag.exe"
	}
	return "nsdiag"
}
