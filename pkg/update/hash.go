package update

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/theupdateframework/go-tuf/data"
)

// CheckFileHash checks the file at the local path against the provided hash
// functions.
func CheckFileHash(meta *data.TargetFileMeta, localPath string) error {
	hashFunc, hashVal, err := selectHashFunction(meta)
	if err != nil {
		return err
	}

	f, err := os.Open(localPath)
	if err != nil {
		return errors.Wrap(err, "open file for hash")
	}

	if _, err := io.Copy(hashFunc, f); err != nil {
		return errors.Wrap(err, "read file for hash")
	}

	if !bytes.Equal(hashVal, hashFunc.Sum(nil)) {
		return errors.Errorf("hash %s does not match expected: %s", data.HexBytes(hashFunc.Sum(nil)), data.HexBytes(hashVal))
	}

	return nil
}

// selectHashFunction returns the first matching hash function and expected
// hash, otherwise returning an error if not matching hash can be found.
func selectHashFunction(meta *data.TargetFileMeta) (hash.Hash, []byte, error) {
	for hashName, hashVal := range meta.Hashes {
		switch hashName {
		case "sha512":
			return sha512.New(), hashVal, nil
		case "sha256":
			return sha256.New(), hashVal, nil
		}
	}

	return nil, nil, errors.Errorf("no matching hash function found: %v", meta.HashAlgorithms())
}
