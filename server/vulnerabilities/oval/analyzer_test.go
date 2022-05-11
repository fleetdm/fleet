package oval

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOvalAnalyzer(t *testing.T) {
	t.Run("#load", func(t *testing.T) {
		t.Run("invalid vuln path", func(t *testing.T) {
			platform := NewPlatform("ubuntu", "Ubuntu 20.4.0")
			_, err := load(platform, "")
			require.Error(t, err, "invalid vulnerabity path")
		})
	})

	t.Run("#latestOvalDefFor", func(t *testing.T) {
		t.Run("definition matching platform for date exists", func(t *testing.T) {
			path, err := ioutil.TempDir("", "oval_test")
			defer os.RemoveAll(path)
			require.NoError(t, err)

			today := time.Now()
			platform := NewPlatform("ubuntu", "Ubuntu 20.4.0")
			def := filepath.Join(path, platform.ToFilename(today, "json"))

			f1, err := os.Create(def)
			require.NoError(t, err)
			f1.Close()

			result, err := latestOvalDefFor(platform, path, today)
			require.NoError(t, err)
			require.Equal(t, def, result)
		})

		t.Run("definition matching platform exists but not for date", func(t *testing.T) {
			path, err := ioutil.TempDir("", "oval_test")
			defer os.RemoveAll(path)
			require.NoError(t, err)

			today := time.Now()
			yesterday := today.Add(-24 * time.Hour)

			platform := NewPlatform("ubuntu", "Ubuntu 20.4.0")
			def := filepath.Join(path, platform.ToFilename(yesterday, "json"))

			f1, err := os.Create(def)
			require.NoError(t, err)
			f1.Close()

			result, err := latestOvalDefFor(platform, path, today)
			require.NoError(t, err)
			require.Equal(t, def, result)
		})

		t.Run("definition does not exists for platform", func(t *testing.T) {
			path, err := ioutil.TempDir("", "oval_test")
			defer os.RemoveAll(path)
			require.NoError(t, err)

			today := time.Now()

			platform1 := NewPlatform("ubuntu", "Ubuntu 20.4.0")
			def1 := filepath.Join(path, platform1.ToFilename(today, "json"))
			f1, err := os.Create(def1)
			require.NoError(t, err)
			f1.Close()

			platform2 := NewPlatform("ubuntu", "Ubuntu 18.4.0")

			_, err = latestOvalDefFor(platform2, path, today)
			require.Error(t, err, "file not found for platform")
		})
	})
}
