//go:build windows

package homes

import "os"

// statOwnerUID has no portable implementation on Windows (ownership is
// expressed via security descriptors / SIDs, not a numeric uid), so callers
// fall back to name-based attribution there.
func statOwnerUID(_ os.FileInfo) (string, bool) { return "", false }
