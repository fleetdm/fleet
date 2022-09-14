package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLatestFile(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		dir := t.TempDir()

		today := time.Now()
		fileName := fmt.Sprintf("file1-%d_%02d_%02d.%s", today.Year(), today.Month(), today.Day(), "json")

		f1, err := os.Create(filepath.Join(dir, fileName))
		require.NoError(t, err)
		f1.Close()

		result, err := LatestFile(fileName, dir)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, fileName), result)
	})

	t.Run("file exists but not for date", func(t *testing.T) {
		dir := t.TempDir()

		today := time.Now()
		yesterday := today.Add(-24 * time.Hour)

		todayFile := fmt.Sprintf("file1-%d_%02d_%02d.%s", today.Year(), today.Month(), today.Day(), "json")
		yesterdayFile := fmt.Sprintf("file1-%d_%02d_%02d.%s", yesterday.Year(), yesterday.Month(), yesterday.Day(), "json")

		f1, err := os.Create(filepath.Join(dir, yesterdayFile))
		require.NoError(t, err)
		f1.Close()

		result, err := LatestFile(todayFile, dir)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, yesterdayFile), result)
	})

	t.Run("file does not exists", func(t *testing.T) {
		dir := t.TempDir()

		today := time.Now()

		wantedFile := fmt.Sprintf("file1-%d_%02d_%02d.%s", today.Year(), today.Month(), today.Day(), "json")
		existingFile := fmt.Sprintf("file2-%d_%02d_%02d.%s", today.Year(), today.Month(), today.Day(), "json")

		f1, err := os.Create(filepath.Join(dir, existingFile))
		require.NoError(t, err)
		f1.Close()

		_, err = LatestFile(wantedFile, dir)
		require.Error(t, err, "file not found")
	})
}
