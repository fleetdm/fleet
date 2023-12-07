package update

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/go-tuf/data"
)

func createFile(t *testing.T, name string, length int) (string, *data.TargetFileMeta) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	require.NoError(t, err)
	dir := t.TempDir()
	filePath := filepath.Join(dir, name)
	err = os.WriteFile(filePath, b, constant.DefaultFileMode)
	require.NoError(t, err)
	sha256Bytes := sha256.Sum256(b)
	sha512Bytes := sha512.Sum512(b)
	return filePath, &data.TargetFileMeta{
		FileMeta: data.FileMeta{
			Length: int64(length),
			Hashes: data.Hashes{
				"sha256": data.HexBytes(sha256Bytes[:]),
				"sha512": data.HexBytes(sha512Bytes[:]),
			},
		},
	}
}

func TestCheckFileHash(t *testing.T) {
	localPath, meta := createFile(t, "test.txt", 256)
	err := checkFileHash(meta, localPath)
	require.NoError(t, err)

	localPath2, _ := createFile(t, "test2.txt", 256)
	err = checkFileHash(meta, localPath2)
	require.Error(t, err)

	delete(meta.Hashes, "sha512")

	err = checkFileHash(meta, localPath)
	require.NoError(t, err)

	delete(meta.Hashes, "sha256")

	err = checkFileHash(meta, localPath)
	require.Error(t, err)
}

func TestSelectHashFunction(t *testing.T) {
	_, meta := createFile(t, "test.txt", 256)
	hashFn, hashVal, err := selectHashFunction(meta)
	require.NoError(t, err)
	require.Equal(t, hashFn, sha512.New())
	require.Equal(t, hashVal, []byte(meta.Hashes["sha512"]))
	delete(meta.Hashes, "sha512")
	hashFn, hashVal, err = selectHashFunction(meta)
	require.NoError(t, err)
	require.Equal(t, hashFn, sha256.New())
	require.Equal(t, hashVal, []byte(meta.Hashes["sha256"]))
	delete(meta.Hashes, "sha256")
	_, _, err = selectHashFunction(meta)
	require.Error(t, err)
}
