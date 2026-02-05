package server

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/db"
)

// Start starts CVE dictionary HTTP Server.
func Start(logToFile bool, logDir string, driver db.DB) error {
	e := echo.New()
	e.Debug = viper.GetBool("debug")

	// Middleware
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: os.Stderr}))
	e.Use(middleware.Recover())

	// setup access logger
	if logToFile {
		logPath := filepath.Join(logDir, "access.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return xerrors.Errorf("Failed to open a log file. err: %w", err)
		}
		defer f.Close()
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: f}))
	}

	// Routes
	e.GET("/health", health())
	e.GET("/packs/:family/:release/:pack/:arch", getByPackName(driver))
	e.GET("/packs/:family/:release/:pack", getByPackName(driver))
	e.GET("/cves/:family/:release/:id/:arch", getByCveID(driver))
	e.GET("/cves/:family/:release/:id", getByCveID(driver))
	e.GET("/advisories/:family/:release", getAdvisories(driver))
	e.GET("/count/:family/:release", countOvalDefs(driver))
	e.GET("/lastmodified/:family/:release", getLastModified(driver))
	//  e.Post("/cpes", getByPackName(driver))

	bindURL := fmt.Sprintf("%s:%s", viper.GetString("bind"), viper.GetString("port"))
	log15.Info("Listening...", "URL", bindURL)
	return e.Start(bindURL)
}

// Handler
func health() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "")
	}
}

func getByPackName(driver db.DB) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		family := strings.ToLower(c.Param("family"))
		release := c.Param("release")
		pack := c.Param("pack")
		arch := c.Param("arch")
		decodePack, err := url.QueryUnescape(pack)
		if err != nil {
			log15.Error(fmt.Sprintf("Failed to Decode Package Name: %s", err))
			return c.JSON(http.StatusBadRequest, nil)
		}

		log15.Debug("Params", "Family", family, "Release", release, "Pack", pack, "DecodePack", decodePack, "arch", arch)

		defs, err := driver.GetByPackName(family, release, decodePack, arch)
		if err != nil {
			log15.Error("Failed to get by Package Name.", "err", err)
		}
		return c.JSON(http.StatusOK, defs)
	}
}

func getByCveID(driver db.DB) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		family := strings.ToLower(c.Param("family"))
		release := c.Param("release")
		cveID := c.Param("id")
		arch := c.Param("arch")
		log15.Debug("Params", "Family", family, "Release", release, "CveID", cveID, "arch", arch)

		defs, err := driver.GetByCveID(family, release, cveID, arch)
		if err != nil {
			log15.Error("Failed to get by CveID.", "err", err)
		}
		return c.JSON(http.StatusOK, defs)
	}
}

func getAdvisories(driver db.DB) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		family := strings.ToLower(c.Param("family"))
		release := c.Param("release")
		log15.Debug("Params", "Family", family, "Release", release)

		m, err := driver.GetAdvisories(family, release)
		if err != nil {
			log15.Error("Failed to get advisories.", "err", err)
		}
		return c.JSON(http.StatusOK, m)
	}
}

func countOvalDefs(driver db.DB) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		family := strings.ToLower(c.Param("family"))
		release := c.Param("release")
		log15.Debug("Params", "Family", family, "Release", release)

		count, err := driver.CountDefs(family, release)
		if err != nil {
			log15.Error("Failed to count OVAL defs.", "err", err)
		}
		return c.JSON(http.StatusOK, count)
	}
}

func getLastModified(driver db.DB) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		family := strings.ToLower(c.Param("family"))
		release := c.Param("release")
		log15.Debug("Params", "Family", family, "Release", release)

		t, err := driver.GetLastModified(family, release)
		if err != nil {
			log15.Error(fmt.Sprintf("Failed to GetLastModified: %s", err))
			return c.JSON(http.StatusInternalServerError, nil)
		}

		return c.JSON(http.StatusOK, t)
	}
}
