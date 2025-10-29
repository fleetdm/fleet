package update

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/theupdateframework/go-tuf/data"
)

// checkFileHash checks the file at the local path against the provided hash functions.
func checkFileHash(meta *data.TargetFileMeta, localPath string) error {
	metaHash, localHash, err := fileHashes(meta, localPath)
	if err != nil {
		return fmt.Errorf("failed to calculate local file hash: %s", err)
	}
	if !bytes.Equal(localHash, metaHash) {
		return fmt.Errorf("hash %x does not match expected: %x", localHash, metaHash)
	}
	return nil
}

func fileHashes(meta *data.TargetFileMeta, localPath string) (metaHash []byte, localHash []byte, err error) {
	hashFn, metaHash, err := selectHashFunction(meta)
	if err != nil {
		return nil, nil, err
	}

	f, err := os.Open(localPath)
	if err != nil {
		// If tar.gz doesn't exist but a hash file does, use the cached hash file
		if os.IsNotExist(err) && strings.HasSuffix(localPath, ".tar.gz") {
			cachedHash, err := readCachedHash(localPath, meta)
			if err == nil {
				return metaHash, cachedHash, nil
			}
			log.Info().Err(err).Msg("failed to read cached hash file")
		}
		return nil, nil, fmt.Errorf("open file for hash: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(hashFn, f); err != nil {
		return nil, nil, fmt.Errorf("read file for hash: %w", err)
	}
	return metaHash, hashFn.Sum(nil), nil
}

// selectHashFunction returns the first matching hash function and expected
// hash, otherwise returning an error if no matching hash can be found.
//
// SHA512 is preferred, and SHA256 is returned if 512 is not available.
func selectHashFunction(meta *data.TargetFileMeta) (hash.Hash, []byte, error) {
	for hashName, hashVal := range meta.Hashes {
		if hashName == "sha512" {
			return sha512.New(), hashVal, nil
		}
	}

	for hashName, hashVal := range meta.Hashes {
		if hashName == "sha256" {
			return sha256.New(), hashVal, nil
		}
	}

	return nil, nil, fmt.Errorf("no matching hash function found: %v", meta.HashAlgorithms())
}

// readCachedHash reads a cached hash from a .sha512 file
// created during packaging when the tar.gz was removed to save space.
func readCachedHash(tarGzPath string, meta *data.TargetFileMeta) ([]byte, error) {
	// Check if TUF metadata has SHA512 (currently the only hash file used)
	for hashName := range meta.Hashes {
		if hashName == "sha512" {
			hashPath := tarGzPath + ".sha512"
			var hashHex []byte
			var err error
			if hashHex, err = os.ReadFile(hashPath); err != nil {
				return nil, err
			}
			return hex.DecodeString(strings.TrimSpace(string(hashHex)))
		}
	}

	return nil, fmt.Errorf("no cached hash file found for %s", tarGzPath)
}
