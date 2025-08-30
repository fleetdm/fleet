package main

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupAndRestore(t *testing.T) {
	// Init a mock input directory.
	mockInput := t.TempDir()
	t.Logf("using mock input directory [%s]", mockInput)

	// Populate the input directory with some fake files.
	const numFiles = 128           // Number of fake files to create.
	const fileNameLen = 32         // Length of the randomize file name (32-bytes).
	const fileSizeMin = 64         // Minimum file size to create (64-bytes).
	const fileSizeMax = 500 * 1024 // Maximum file size to create (500-kilobytes).
	files := rndFiles(t, mockInput, numFiles, fileNameLen, fileSizeMin, fileSizeMax)

	// Confirm we generated the expected number of files.
	require.Len(t, files, numFiles)

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
		// Base-16-encode the hash.
		hash := hex.EncodeToString(hashSum[:])

		// Ensure the hashes match.
		require.Equal(t, file.Hash, hash)
	}
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
	fileName := rndString(t, fileNameLen)

	// Generate a random number of nested path segments.
	const pathSegMin = 0
	const pathSegMax = 4
	const pathSegLenMin = 8
	const pathSegLenMax = 16
	//nolint:gosec,G304 // We don't need complex randoms.
	numPathSegs := pathSegMin + rand.IntN(pathSegMax-pathSegMin)
	pathSegs := make([]string, numPathSegs, numPathSegs+2)
	for i := range pathSegs {
		// Generate a random path segment length.
		//
		//nolint:gosec,G304 // We don't need complex randoms.
		pathSegLen := pathSegLenMin + rand.IntN(pathSegLenMax-pathSegLenMin)
		pathSegs[i] = rndString(t, pathSegLen)
	}

	// Construct the parent directory path.
	pathSegs = append([]string{path}, pathSegs...)
	pathSegs = append(pathSegs, fileName)
	filePath := filepath.Join(pathSegs...)

	// Create the parent directory structure.
	err := os.MkdirAll(filepath.Dir(filePath), fileModeUserRWX)
	require.NoError(t, err)

	// Generate random file content.
	content := rndFileContent(t, fileSizeMin, fileSizeMax)

	// Create and get a writable handle to the new file.
	//
	//nolint:gosec,G304 // 'filePath' is a trusted input.
	f, err := os.Create(filePath)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	// Init the SHA-256 hasher.
	hasher := sha256.New()

	// Wrap the hasher + file in a multiwriter.
	w := io.MultiWriter(f, hasher)

	// Write the file content to the hasher + file.
	n, err := w.Write(content)
	require.NoError(t, err)
	require.Equal(t, len(content), n)

	// Sum and base-16 encode the SHA-256 hash.
	hashSum := hasher.Sum(nil)
	hash := hex.EncodeToString(hashSum)

	return File{
		Path: filePath,
		Hash: hash,
	}
}

// rndString generates a 'length'-length random string using the hexadecimal
// character set (a-f, 0-9).
func rndString(t *testing.T, length int) string {
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
	//nolint:gosec,G304 // We don't need complex randoms.
	size := sizeMin + rand.IntN(sizeMax-sizeMin)

	// Init the "file" byte slice and fill it with random data.
	file := make([]byte, size)
	n, err := crand.Reader.Read(file)
	require.Equal(t, size, n)
	require.NoError(t, err)

	return file
}
