// Package main contains an executable that migrates all targets from one source TUF repository
// to a destination TUF repository. It migrates all targets except a few known unused targets.
package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	if runtime.GOOS == "windows" {
		log.Fatalf("%s is not supported on windows", os.Args[0])
	}

	sourceRepositoryDirectory := flag.String("source-repository-directory", "", "Absolute path directory for the source TUF")
	destRepositoryDirectory := flag.String("dest-repository-directory", "", "Absolute path directory for the destination TUF")

	flag.Parse()

	if *sourceRepositoryDirectory == "" {
		log.Fatal("missing --source-repository-directory")
	}
	if *destRepositoryDirectory == "" {
		log.Fatal("missing --dest-repository-directory")
	}

	type targetEntry struct {
		sha512 string
		length int
	}

	// Perform addition of targets by iterating source repository.
	sourceEntries := make(map[string]targetEntry)
	iterateRepository(*sourceRepositoryDirectory, func(target, targetPath, platform, targetName, version, channel, hashSHA512 string, length int) error {
		cmd := exec.Command("fleetctl", "updates", "add", //nolint:gosec
			"--path", *destRepositoryDirectory,
			"--target", targetPath,
			"--platform", platform,
			"--name", targetName,
			"--version", version,
			"-t", channel,
		)
		log.Print(cmd.String())
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		if err := cmd.Run(); err != nil {
			log.Fatalf("target: %q: failed to add target: %s", target, err)
		}

		sourceEntries[target] = targetEntry{
			sha512: hashSHA512,
			length: length,
		}

		return nil
	})

	// Perform validation of destination repository.
	iterateRepository(*destRepositoryDirectory, func(target, targetPath, platform, targetName, version, channel, hashSHA512 string, length int) error {
		sourceEntry, ok := sourceEntries[target]
		if !ok {
			return errors.New("entry not found in source directory")
		}

		if sourceEntry.length != length {
			return fmt.Errorf("mismatch length: %d vs %d", length, sourceEntry.length)
		}
		if sourceEntry.sha512 != hashSHA512 {
			return fmt.Errorf("mismatch sha512: %s vs %s", hashSHA512, sourceEntry.sha512)
		}

		targetBytes, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		h := sha512.New()
		if _, err := h.Write(targetBytes); err != nil {
			return fmt.Errorf("failed to hash file: %w", err)
		}
		fileHash := hex.EncodeToString(h.Sum(nil))
		if fileHash != sourceEntry.sha512 {
			return fmt.Errorf("mismatch sha512 and file contents: %s vs %s", fileHash, sourceEntry.sha512)
		}

		return nil
	})
}

func iterateRepository(repositoryDirectory string, fn func(target, targetPath, platform, targetName, version, channel, sha512 string, length int) error) {
	repositoryPath := filepath.Join(repositoryDirectory, "repository")
	targetsFile := filepath.Join(repositoryPath, "targets.json")
	targetsBytes, err := os.ReadFile(targetsFile)
	if err != nil {
		log.Fatal("failed to read the source targets.json file")
	}

	var targetsJSON map[string]interface{}
	if err := json.Unmarshal(targetsBytes, &targetsJSON); err != nil {
		log.Fatal("failed to parse the source targets.json file")
	}

	signed_ := targetsJSON["signed"]
	if signed_ == nil {
		log.Fatal("missing signed key in targets.json file")
	}
	signed, ok := signed_.(map[string]interface{})
	if !ok {
		log.Fatalf("invalid signed key in targets.json file: %T, expected map", signed_)
	}
	targets_ := signed["targets"]
	if targets_ == nil {
		log.Fatal("missing signed.targets key in targets.json file")
	}
	targets, ok := targets_.(map[string]interface{})
	if !ok {
		log.Fatalf("invalid signed.targets key in targets.json file: %T, expected map", targets_)
	}

	for target, metadata_ := range targets {
		targetPath := filepath.Join(repositoryPath, "targets", target)

		parts := strings.Split(target, "/")
		if len(parts) != 4 {
			log.Fatalf("target %q: invalid number of parts, expected 4", target)
		}

		targetName := parts[0]
		platform := parts[1]
		channel := parts[2]

		// Unused targets (probably accidentally pushed).
		if targetName == "desktop.tar.gz" {
			continue
		}

		metadata, ok := metadata_.(map[string]interface{})
		if !ok {
			log.Fatalf("target: %q: invalid metadata field: %T, expected map", target, metadata_)
		}
		custom_ := metadata["custom"]
		if custom_ == nil {
			log.Fatalf("target: %q: missing custom field", target)
		}
		custom, ok := custom_.(map[string]interface{})
		if !ok {
			log.Fatalf("target: %q: invalid custom field: %T, expected map", target, custom_)
		}
		version_ := custom["version"]
		if version_ == nil {
			log.Fatalf("target: %q: missing custom.version field", target)
		}
		version, ok := version_.(string)
		if !ok {
			log.Fatalf("target: %q: invalid custom.version field: %T", target, version_)
		}
		length_ := metadata["length"]
		if length_ == nil {
			log.Fatalf("target: %q: missing length field", target)
		}
		lengthf, ok := length_.(float64)
		if !ok {
			log.Fatalf("target: %q: invalid length field: %T", target, length_)
		}
		length := int(lengthf)
		hashes_ := metadata["hashes"]
		if hashes_ == nil {
			log.Fatalf("target: %q: missing hashes field", target)
		}
		hashes, ok := hashes_.(map[string]interface{})
		if !ok {
			log.Fatalf("target: %q: invalid hashes field: %T", target, hashes_)
		}
		sha512_ := hashes["sha512"]
		if sha512_ == nil {
			log.Fatalf("target: %q: missing hashes.sha512 field", target)
		}
		hashSHA512, ok := sha512_.(string)
		if !ok {
			log.Fatalf("target: %q: invalid hashes.sha512 field: %T", target, sha512_)
		}

		if err := fn(target, targetPath, platform, targetName, version, channel, hashSHA512, length); err != nil {
			log.Fatalf("target: %q: failed to process target: %s", target, err)
		}
	}
}
