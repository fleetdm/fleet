//go:build darwin

package adobe_plugins

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

func getScanPaths(level string, logger zerolog.Logger) ([]scanPath, error) {
	var paths []scanPath

	// System-wide CEP extensions
	paths = append(paths, scanPath{
		basePath:      "/Library/Application Support/Adobe/CEP/extensions",
		extensionType: "CEP",
	})
	// System-wide UXP extensions
	paths = append(paths, scanPath{
		basePath:      "/Library/Application Support/Adobe/UXP/extensions",
		extensionType: "UXP",
	})

	// Per-user CEP and UXP extensions
	users, err := listLocalUsers()
	if err != nil {
		logger.Warn().Err(err).Msg("failed to enumerate local users, skipping per-user paths")
	}
	for _, u := range users {
		paths = append(paths, scanPath{
			basePath:      filepath.Join(u.homeDir, "Library", "Application Support", "Adobe", "CEP", "extensions"),
			extensionType: "CEP",
			user:          u.name,
		})
		paths = append(paths, scanPath{
			basePath:      filepath.Join(u.homeDir, "Library", "Application Support", "Adobe", "UXP", "extensions"),
			extensionType: "UXP",
			user:          u.name,
		})
	}

	if level == "deep" {
		paths = append(paths,
			scanPath{
				basePath:      "/Applications/Adobe Photoshop */Plug-ins",
				extensionType: "native",
				hostApp:       "Photoshop",
			},
			scanPath{
				basePath:      "/Applications/Adobe Premiere Pro */Plug-ins",
				extensionType: "native",
				hostApp:       "Premiere Pro",
			},
			scanPath{
				basePath:      "/Applications/Adobe After Effects */Plug-ins",
				extensionType: "native",
				hostApp:       "After Effects",
			},
			scanPath{
				basePath:      "/Applications/Adobe Illustrator */Plug-ins",
				extensionType: "native",
				hostApp:       "Illustrator",
			},
		)
	}

	return paths, nil
}

type localUser struct {
	name    string
	homeDir string
}

func listLocalUsers() ([]localUser, error) {
	entries, err := os.ReadDir("/Users")
	if err != nil {
		return nil, err
	}

	var users []localUser
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "Shared" {
			continue
		}
		users = append(users, localUser{
			name:    name,
			homeDir: filepath.Join("/Users", name),
		})
	}
	return users, nil
}
