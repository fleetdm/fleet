package table

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type findFile struct {
	username string
}

type FindFileOpt func(*findFile)

func WithUsername(username string) FindFileOpt {
	return func(ff *findFile) {
		ff.username = username
	}
}

var homeDirLocations = map[string][]string{
	"windows": {"/Users"}, // windows10 uses /Users
	"darwin":  {"/Users"},
}
var homeDirDefaultLocation = []string{"/home"}

type userFileInfo struct {
	user string
	path string
}

// findFileInUserDirs looks for the existence of a specified path as a
// subdirectory of users' home directories. It does this by searching
// likely paths
func findFileInUserDirs(pattern string, logger log.Logger, opts ...FindFileOpt) ([]userFileInfo, error) {
	ff := &findFile{}

	for _, opt := range opts {
		opt(ff)
	}

	homedirRoots, ok := homeDirLocations[runtime.GOOS]
	if !ok {
		homedirRoots = homeDirDefaultLocation
		level.Debug(logger).Log(
			"msg", "platform not found using default",
			"homeDirRoot", homedirRoots,
		)
	}

	foundPaths := []userFileInfo{}

	// Redo/remove when we make username a required parameter
	if ff.username == "" {
		for _, possibleHome := range homedirRoots {

			userDirs, err := os.ReadDir(possibleHome)
			if err != nil {
				// This possibleHome doesn't exist. Move on
				continue
			}

			// For each user's dir, in this possibleHome, check!
			for _, ud := range userDirs {
				userPathPattern := filepath.Join(possibleHome, ud.Name(), pattern)
				fullPaths, err := filepath.Glob(userPathPattern)
				if err != nil {
					// skipping ErrBadPattern
					level.Debug(logger).Log(
						"msg", "bad file pattern",
						"pattern", userPathPattern,
					)
					continue
				}
				for _, fullPath := range fullPaths {
					if stat, err := os.Stat(fullPath); err == nil && stat.Mode().IsRegular() {
						foundPaths = append(foundPaths, userFileInfo{
							user: ud.Name(),
							path: fullPath,
						})
					}
				}
			}
		}

		return foundPaths, nil
	}

	// We have a username. Future normal path here
	for _, possibleHome := range homedirRoots {
		userPathPattern := filepath.Join(possibleHome, ff.username, pattern)
		fullPaths, err := filepath.Glob(userPathPattern)
		if err != nil {
			// skipping ErrBadPattern
			level.Debug(logger).Log(
				"msg", "bad file pattern",
				"pattern", userPathPattern,
			)
			continue
		}
		for _, fullPath := range fullPaths {
			if stat, err := os.Stat(fullPath); err == nil && stat.Mode().IsRegular() {
				foundPaths = append(foundPaths, userFileInfo{
					user: ff.username,
					path: fullPath,
				})
			}
		}
	}
	return foundPaths, nil
}

func btoi(value bool) int {
	if value {
		return 1
	}
	return 0
}
