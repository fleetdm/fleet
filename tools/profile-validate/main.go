// Command profile-validate checks Apple profiles and declarations and Windows profiles against the
// embedded profile validation schema, for running ad hoc over the repo's profiles.
//
//	go run ./tools/profile-validate [-strict] [paths...]
//
// Paths are files or directories and default to the directories that hold Fleet's own profiles.
// It prints one line per finding and exits 1 if there are any.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
)

var defaultPaths = []string{"it-and-security", "docs/solutions", "ee/cis"}

func main() {
	strict := flag.Bool("strict", false, "also report unknown types, keys, and LocURIs, envelope problems, and Windows format and scope disagreements")
	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	paths := flag.Args()
	if len(paths) == 0 {
		paths = defaultPaths
	}
	if err := profilevalidation.ParseProfileValidationSchema(); err != nil {
		logger.ErrorContext(context.Background(), "parse embedded schema", "err", err)
		os.Exit(2)
	}
	mode := profilevalidation.ModeLenient
	if *strict {
		mode = profilevalidation.ModeStrict
	}
	findings, err := run(context.Background(), logger, os.Stdout, profilevalidation.Current(), mode, paths)
	switch {
	case err != nil:
		logger.ErrorContext(context.Background(), "validate", "err", err)
		os.Exit(2)
	case findings > 0:
		os.Exit(1)
	}
}

// run validates every profile and declaration under paths, writes one line per finding to
// out, and returns how many findings there were. A file that can't be read as a profile is logged
// and skipped: the repo has profiles with GitOps environment variables that fleetctl expands
// before upload.
func run(ctx context.Context, logger *slog.Logger, out io.Writer, s *profilevalidation.ProfileValidationSchema, mode profilevalidation.Mode, paths []string) (int, error) {
	var files, findings, skipped int
	for _, arg := range paths {
		info, err := os.Stat(arg)
		if err != nil {
			return findings, err
		}
		// Walk inside an os.Root so a symlink can't lead the reads outside the given path.
		dir, name := arg, "."
		if !info.IsDir() {
			dir, name = filepath.Dir(arg), filepath.Base(arg)
		}
		root, err := os.OpenRoot(dir)
		if err != nil {
			return findings, err
		}
		fsys := root.FS()
		err = fs.WalkDir(fsys, name, func(rel string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			path := filepath.Join(dir, rel)
			validate := validatorFor(s, path)
			if validate == nil {
				return nil
			}
			data, err := fs.ReadFile(fsys, rel)
			if err != nil {
				return err
			}
			got, err := validate(data, mode)
			if errors.Is(err, errNotProfile) {
				return nil
			}
			files++
			if err != nil {
				skipped++
				logger.WarnContext(ctx, "skipped unreadable file", "path", path, "err", err)
				return nil
			}
			for _, f := range got {
				findings++
				line := fmt.Sprintf("%s: %s %s", path, f.Code, f.PayloadType)
				if f.KeyPath != "" {
					line += " " + f.KeyPath
				}
				if len(f.Expected) > 0 {
					line += fmt.Sprintf(" expected=%s actual=%s", strings.Join(f.Expected, "|"), f.Actual)
				}
				if _, err := fmt.Fprintln(out, line); err != nil {
					return err
				}
			}
			return nil
		})
		closeErr := root.Close()
		if closeErr != nil {
			return findings, closeErr
		}
		if err != nil {
			return findings, err
		}
	}
	logger.InfoContext(ctx, "validated", "files", files, "findings", findings, "skipped", skipped)
	return findings, nil
}

var errNotProfile = errors.New("not a profile")

// validatorFor picks the validator from the file name: .mobileconfig is an Apple profile, .json a
// declaration unless its Type isn't com.apple.* (the same trees hold Android and other JSON), and
// .xml a Windows profile if it has a <LocURI> (the same trees hold other XML).
func validatorFor(s *profilevalidation.ProfileValidationSchema, path string) func([]byte, profilevalidation.Mode) ([]profilevalidation.Finding, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mobileconfig":
		return s.ValidateAppleProfile
	case ".json":
		return func(data []byte, mode profilevalidation.Mode) ([]profilevalidation.Finding, error) {
			var head struct {
				Type any `json:"Type"`
			}
			if json.Unmarshal(data, &head) != nil {
				// Could be a broken declaration: let the validator's decode error get it logged as skipped.
				return s.ValidateAppleDeclaration(data, mode)
			}
			if t, _ := head.Type.(string); !strings.HasPrefix(t, "com.apple.") {
				return nil, errNotProfile
			}
			return s.ValidateAppleDeclaration(data, mode)
		}
	case ".xml":
		return func(data []byte, mode profilevalidation.Mode) ([]profilevalidation.Finding, error) {
			decoder := xml.NewDecoder(bytes.NewReader(data))
			hasLocURI := false
			for {
				token, err := decoder.Token()
				if err != nil {
					break
				}
				if element, ok := token.(xml.StartElement); ok && element.Name.Local == "LocURI" {
					hasLocURI = true
					break
				}
			}
			if !hasLocURI {
				return nil, errNotProfile
			}
			return s.ValidateWindowsProfile(data, mode)
		}
	}
	return nil
}
