//go:build windows

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
		basePath:      `C:\Program Files\Common Files\Adobe\CEP\extensions`,
		extensionType: "CEP",
	})
	paths = append(paths, scanPath{
		basePath:      `C:\Program Files (x86)\Common Files\Adobe\CEP\extensions`,
		extensionType: "CEP",
	})

	// System-wide UXP extensions
	paths = append(paths, scanPath{
		basePath:      `C:\Program Files\Common Files\Adobe\UXP\extensions`,
		extensionType: "UXP",
	})

	// Per-user CEP and UXP extensions
	users, err := listLocalUsers()
	if err != nil {
		logger.Warn().Err(err).Msg("failed to enumerate local users, skipping per-user paths")
	}
	for _, u := range users {
		paths = append(paths, scanPath{
			basePath:      filepath.Join(u.homeDir, "AppData", "Roaming", "Adobe", "CEP", "extensions"),
			extensionType: "CEP",
			user:          u.name,
		})
		paths = append(paths, scanPath{
			basePath:      filepath.Join(u.homeDir, "AppData", "Roaming", "Adobe", "UXP", "extensions"),
			extensionType: "UXP",
			user:          u.name,
		})
	}

	if level == "deep" {
		paths = append(paths,
			scanPath{
				basePath:      `C:\Program Files\Adobe\Adobe Photoshop *\Plug-ins`,
				extensionType: "native",
				hostApp:       "Photoshop",
			},
			scanPath{
				basePath:      `C:\Program Files\Adobe\Adobe Premiere Pro *\Plug-ins`,
				extensionType: "native",
				hostApp:       "Premiere Pro",
			},
			scanPath{
				basePath:      `C:\Program Files\Adobe\Adobe After Effects *\Plug-ins`,
				extensionType: "native",
				hostApp:       "After Effects",
			},
		)
	}

	return paths, nil
}

func listLocalUsers() ([]localUser, error) {
	entries, err := os.ReadDir(`C:\Users`)
	if err != nil {
		return nil, err
	}

	skipNames := map[string]struct{}{
		"public":       {},
		"default":      {},
		"default user": {},
		"all users":    {},
	}

	var users []localUser
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if _, skip := skipNames[strings.ToLower(name)]; skip {
			continue
		}
		users = append(users, localUser{
			name:    name,
			homeDir: filepath.Join(`C:\Users`, name),
		})
	}
	return users, nil
}
