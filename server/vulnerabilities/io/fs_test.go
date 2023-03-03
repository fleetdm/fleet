package io

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFSClient(t *testing.T) {
	t.Run("#Bulletins", func(t *testing.T) {
		t.Run("directory does not exists", func(t *testing.T) {
			sut := NewFSClient("asdf")
			_, err := sut.MSRCBulletins()
			require.Error(t, err)
		})

		t.Run("returns a list of file matching the MSRC file prefix", func(t *testing.T) {
			path := t.TempDir()
			sut := NewFSClient(path)

			file1 := filepath.Join(path, "my_lyrics-2021_10_10.json")
			bulletin1 := filepath.Join(path, fmt.Sprintf("%sWindows_10-2022_10_10.json", mSRCFilePrefix))
			bulletin2 := filepath.Join(path, fmt.Sprintf("%sWindows_11-2022_10_10.json", mSRCFilePrefix))

			f1, err := os.Create(bulletin1)
			require.NoError(t, err)
			f1.Close()

			f2, err := os.Create(bulletin2)
			require.NoError(t, err)
			f2.Close()

			f3, err := os.Create(file1)
			require.NoError(t, err)
			f3.Close()

			r, err := sut.MSRCBulletins()
			require.NoError(t, err)

			a, err := NewMSRCMetadata(filepath.Base(file1))
			require.NoError(t, err)
			b, err := NewMSRCMetadata(filepath.Base(bulletin1))
			require.NoError(t, err)
			c, err := NewMSRCMetadata(filepath.Base(bulletin2))
			require.NoError(t, err)

			require.NotContains(t, r, a)
			require.Contains(t, r, b)
			require.Contains(t, r, c)
		})
	})
}
