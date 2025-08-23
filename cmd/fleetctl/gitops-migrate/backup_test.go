package main

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"math/big"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackup(t *testing.T) {
	// Init a mock input directory.
	mockInput := t.TempDir()
	t.Logf("using mock input directory [%s]", mockInput)

	// Populate the input directory with some fake files.
	const numFiles = 8             // Number of fake files to create.
	const fileNameLen = 32         // Length of the randomize file name (32-bytes).
	const fileSizeMin = 64         // Minimum file size to create (64-bytes).
	const fileSizeMax = 500 * 1024 // Maximum file size to create (500-kilobytes).
	files := rndFiles(t, mockInput, numFiles, fileNameLen, fileSizeMin, fileSizeMax)

	// Init the mock output directory (destination for the backup).
	mockOutput := t.TempDir()
	t.Logf("using mock output directory [%s]", mockOutput)

	// Test backup and restore.
	testBackupAndRestore(t, mockInput, mockOutput, files)
}

func testBackupAndRestore(t *testing.T, from, to string, files []File) {
	t.Helper()

	ctx := t.Context()

	// Perform the backup.
	archivePath, err := backup(ctx, from, to)
	require.NoError(t, err)
	require.NotEmpty(t, archivePath)

	// Now untar what we wrote to disk.
	require.NoError(t, restore(ctx, archivePath, to))

	// Remove the archive.
	require.NoError(t, os.Remove(archivePath))

	// Mutate 'from' as-needed.
	//
	// NOTE: This is only required for the test case where 'from' points to a
	// file. This is required since we need to strip the 'from' prefix in the loop
	// below to check the 'to'-side file existence and hash. There's also no
	// trouble in us mutating it here as the condition we're testing (by passing
	// the file path) occurs in the call to 'backup' above.
	stats, err := os.Stat(from)
	require.NoError(t, err)
	require.NotNil(t, stats)
	if !stats.IsDir() {
		from = filepath.Dir(from)
	}

	// Recursively iterate all _input_ files, transposed to the 'mockOutput' dir,
	// hash their contents and ensure it matches what we expect.
	for _, file := range files {
		// Replace the 'mockInput' prefix with 'mockOutput' (to mirror to the
		// output directory).
		//
		//nolint:gosec,G304 // 'file.Path' is a trusted input.
		path := strings.TrimPrefix(file.Path, from)
		path = filepath.Join(to, path)

		// Open a readable handle to the file on-disk.
		//
		//nolint:gosec,G304 // 'path' is a trusted input.
		f, err := os.Open(path)
		require.NoError(t, err)

		// Read the entire file content.
		content, err := io.ReadAll(f)
		require.NoError(t, err)

		// Close the file.
		require.NoError(t, f.Close())

		// SHA-256 hash the file contents.
		hashSum := sha256.Sum256(content)
		// Base-16-encode the file contents.
		hash := hex.EncodeToString(hashSum[:])

		// Ensure the hashes match.
		require.Equal(t, file.Hash, hash)
	}
}

type File struct {
	Path string
	Hash string
}

// rndFiles generates 'fileCount' random files in the directory defined by
// 'path', returning the absolute file path and SHA-256 hash for each.
func rndFiles(t *testing.T, path string, fileCount, fileNameLen, fileSizeMin, fileSizeMax int) []File {
	t.Helper()

	// Init a slice to hold the randomly generated files.
	files := make([]File, fileCount)

	// Generate 'fileCount' random files.
	for i := range fileCount {
		files[i] = rndFile(t, path, fileNameLen, fileSizeMin, fileSizeMax)
	}

	return files
}

// rndFile generates a random file in the directory defined by 'path', with a
// randomly generated name of 'fileNameLen' length and randomly generated
// content that is between 'fileSizeMin' and 'fileSizeMax' in size.
//
// The returned file holds the absolute file path and the SHA-256 hash of its
// contents.
func rndFile(t *testing.T, path string, fileNameLen, fileSizeMin, fileSizeMax int) File {
	t.Helper()

	// Randomly generate a file name.
	fileName := rndFileName(t, fileNameLen)

	// Construct the absolute file path.
	filePath := filepath.Join(path, fileName)

	// Generate random file content.
	content := rndFileContent(t, fileSizeMin, fileSizeMax)

	// Create and get a writable handle to the new file.
	//
	//nolint:gosec,G304 // 'filePath' is a trusted input.
	f, err := os.Create(filePath)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	// Wrap the random file content into a 'bytes.Reader' (io.Reader impl.).
	r := bytes.NewReader(content)

	// Init a TeeReader to transparently SHA-256 hash the file content as we write
	// it to disk.
	tr := io.TeeReader(r, f)

	// Init the SHA-256 hasher.
	hasher := sha256.New()

	// Write to disk and hash the random file content.
	n, err := io.Copy(hasher, tr)
	require.Equal(t, int64(len(content)), n)
	require.NoError(t, err)

	// Sum and base-16 encode the SHA-256 hash.
	hashSum := hasher.Sum(nil)
	hash := hex.EncodeToString(hashSum)

	return File{
		Path: filePath,
		Hash: hash,
	}
}

// rndFileName generates a 'length'-length random string using the hexadecimal
// character set (a-f, 0-9).
func rndFileName(t *testing.T, length int) string {
	t.Helper()

	// Init a 'length/2' (base-16) length byte slice and fill it with random data.
	nameData := make([]byte, length/2)
	n, err := crand.Reader.Read(nameData)
	require.Equal(t, length/2, n)
	require.NoError(t, err)

	// Encode the random data using the base-16 charset.
	return hex.EncodeToString(nameData)
}

// rndFileContent generates a blob of random data, of random size, for file
// mocking.
func rndFileContent(t *testing.T, sizeMin, sizeMax int) []byte {
	t.Helper()

	// Randomize file size.
	//
	// Init a 'big.Int' in the half-open range [0,sizeMin-sizeMax).
	base := big.NewInt(int64(sizeMax - sizeMin))
	// Generate a random operand from our 'big.Int'.
	rnd, err := crand.Int(crand.Reader, base)
	require.NoError(t, err)
	require.True(t, rnd.IsInt64())
	// Add back in the minimum (since it's a half-open range) to produce a truly
	// randomized size in-range.
	size := int64(sizeMin) + rnd.Int64()

	// Init the "file" byte slice and fill it with random data.
	file := make([]byte, size)
	n, err := crand.Reader.Read(file)
	require.Equal(t, size, int64(n))
	require.NoError(t, err)

	return file
}
