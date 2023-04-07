package osquery

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/secure"
)

func WriteSecret(secret string, orbitRoot string) error {
	// Enroll secret
	path := filepath.Join(orbitRoot, "secret.txt")
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := os.WriteFile(path, []byte(secret), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
