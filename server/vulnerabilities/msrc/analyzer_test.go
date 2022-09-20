package msrc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	io "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/io"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	"github.com/stretchr/testify/require"
)

func TestAnalyzer(t *testing.T) {
	t.Run("#loadBulletin", func(t *testing.T) {
		op := fleet.OperatingSystem{
			Name:          "Microsoft Windows 11 Enterprise Evaluation",
			Version:       "21H2",
			Arch:          "64-bit",
			KernelVersion: "10.0.22000.795",
			Platform:      "windows",
		}

		t.Run("dir does not exists", func(t *testing.T) {
			bulletin, err := loadBulletin(op, "over_the_rainbow")
			require.Error(t, err)
			require.Nil(t, bulletin)
		})

		t.Run("returns the lastest bulletin", func(t *testing.T) {
			d := time.Now()
			dir := t.TempDir()

			b := parsed.NewSecurityBulletin(op.Name)
			prod := parsed.NewProductFromOS(op)
			b.Products["1235"] = prod

			fileName := io.FileName(b.ProductName, d)
			filePath := filepath.Join(dir, fileName)

			payload, err := json.Marshal(b)
			require.NoError(t, err)

			err = os.WriteFile(filePath, payload, 0o644)
			require.NoError(t, err)

			actual, err := loadBulletin(op, dir)
			require.NoError(t, err)
			require.Equal(t, prod.Name(), actual.ProductName)
		})
	})
}
