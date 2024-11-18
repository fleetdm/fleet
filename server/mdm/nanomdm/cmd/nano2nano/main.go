package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	stdlog "log"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cli"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"

	"github.com/micromdm/nanolib/log/stdlogfmt"
)

// overridden by -ldflags -X
var version = "unknown"

func main() {
	cliStorage := cli.NewStorage()
	flag.Var(&cliStorage.Storage, "storage", "name of storage backend")
	flag.Var(&cliStorage.DSN, "storage-dsn", "data source name (e.g. connection string or path)")
	flag.Var(&cliStorage.Options, "storage-options", "storage backend options")
	var (
		flVersion = flag.Bool("version", false, "print version")
		flDebug   = flag.Bool("debug", false, "log debug messages")
		flURL     = flag.String("url", "", "NanoMDM migration URL")
		flAPIKey  = flag.String("key", "", "NanoMDM API Key")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		return
	}

	logger := stdlogfmt.New(stdlogfmt.WithDebugFlag(*flDebug))

	var skipServer bool
	if *flURL == "" || *flAPIKey == "" {
		logger.Info("msg", "URL or API key not set; not sending server requests")
		skipServer = true
	}
	client := http.DefaultClient

	mdmStorage, err := cliStorage.Parse(logger)
	if err != nil {
		stdlog.Fatal(err)
	}

	checkins := make(chan interface{})
	ctx := context.Background()
	go func() {
		// dispatch to our storage backend to start sending the checkins
		// channel our MDM check-in messages.
		if err := mdmStorage.RetrieveMigrationCheckins(ctx, checkins); err != nil {
			logger.Info(
				"msg", "retrieving migration checkins",
				"err", err,
			)
		}
		close(checkins)
	}()

	// because order matters (a lot) we are purposefully single threaded for now.
	for checkin := range checkins {
		switch v := checkin.(type) {
		case *mdm.Authenticate:
			logger.Info(logsFromEnrollment("Authenticate", &v.Enrollment)...)
			if !skipServer {
				if err := httpPut(client, *flURL, *flAPIKey, v.Raw); err != nil {
					logger.Info("msg", "sending to migration endpoint", "err", err)
				}
			}
		case *mdm.TokenUpdate:
			logger.Info(logsFromEnrollment("TokenUpdate", &v.Enrollment)...)
			if !skipServer {
				if err := httpPut(client, *flURL, *flAPIKey, v.Raw); err != nil {
					logger.Info("msg", "sending to migration endpoint", "err", err)
				}
			}
		case error:
			logger.Info("msg", "receiving checkin", "err", v)
		default:
			logger.Info("msg", "invalid type provided")
		}
	}
}

func logsFromEnrollment(checkin string, e *mdm.Enrollment) []interface{} {
	r := e.Resolved()
	logs := []interface{}{
		"checkin", checkin,
		"device_id", r.DeviceChannelID,
	}
	if r.UserChannelID != "" {
		logs = append(logs, "user_id", r.UserChannelID)
	}
	if e.UserShortName != "" {
		logs = append(logs, "user_short_name", e.UserShortName)
	}
	logs = append(logs, "type", r.Type.String())
	return logs
}

func httpPut(client *http.Client, url string, key string, sendBytes []byte) error {
	if url == "" || key == "" {
		return errors.New("no URL or API key")
	}
	req, err := http.NewRequest("PUT", url, bytes.NewReader(sendBytes))
	if err != nil {
		return err
	}
	req.SetBasicAuth("nanomdm", key)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Check-in Request failed with HTTP status: %d", res.StatusCode)
	}
	return nil
}
