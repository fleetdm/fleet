package main

import (
	"context"
	"flag"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log/stdlogfmt"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/parse"
	depsync "github.com/fleetdm/fleet/v4/server/mdm/nanodep/sync"
)

// overridden by -ldflags -X
var version = "unknown"

const defaultDuration = 30 * time.Minute

func main() {
	var (
		flVersion = flag.Bool("version", false, "print version")
		flDur     = flag.Uint("duration", uint(defaultDuration/time.Second), "duration in seconds between DEP syncs (0 for single sync)")
		flLimit   = flag.Int("limit", 0, "limit fetch and sync calls to this many devices (0 for server default)")
		flDebug   = flag.Bool("debug", false, "log debug messages")
		flADebug  = flag.Bool("debug-assigner", false, "additional debug logging of the device assigner")
		flStorage = flag.String("storage", "file", "storage backend")
		flDSN     = flag.String("storage-dsn", "", "storage data source name")
		flWebhook = flag.String("webhook-url", "", "URL to send requests to")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <DEPname1> [DEPname2 [...]]\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		return
	}

	if len(flag.Args()) < 1 {
		fmt.Fprintf(flag.CommandLine.Output(), "no DEP names provided\n")
		flag.Usage()
		os.Exit(1)
	}

	logger := stdlogfmt.New(stdlog.Default(), *flDebug)

	storage, err := parse.Storage(*flStorage, *flDSN)
	if err != nil {
		logger.Info("msg", "creating storage backend", "err", err)
		os.Exit(1)
	}

	var webhook *Webhook
	if *flWebhook != "" {
		webhook = NewWebhook(*flWebhook)
	}

	ctx, cancelCtx := context.WithCancel(context.Background())

	// we keep an array of channels to broadcast our syncnow signal
	// for each DEP name we're running a syncer for
	var (
		syncNows   []chan<- struct{}
		syncNowsMu sync.RWMutex
	)

	registerSyncNow := func(c chan<- struct{}) {
		defer syncNowsMu.Unlock()
		syncNowsMu.Lock()
		syncNows = append(syncNows, c)
	}

	closeSyncNow := func(c chan<- struct{}) {
		defer close(c)
		defer syncNowsMu.Unlock()
		syncNowsMu.Lock()
		for i, syncNow := range syncNows {
			if syncNow == c {
				// remove c from list by replace-and-truncate
				syncNows[i] = syncNows[len(syncNows)-1]
				syncNows = syncNows[:len(syncNows)-1]
			}
		}
	}

	sendSyncNows := func() {
		defer syncNowsMu.RUnlock()
		syncNowsMu.RLock()
		for _, syncNow := range syncNows {
			go func(c chan<- struct{}) { c <- struct{}{} }(syncNow)
		}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, os.Interrupt, syscall.SIGTERM)

	// signal handler
	go func() {
		for {
			sig := <-signals
			logger.Debug("msg", "signal received", "signal", sig)
			switch sig {
			case syscall.SIGHUP:
				sendSyncNows()
			case os.Interrupt, syscall.SIGTERM:
				cancelCtx()
			}
		}
	}()

	client := godep.NewClient(storage, http.DefaultClient)

	var wg sync.WaitGroup

	for _, name := range flag.Args()[0:] {

		// create the assigner
		assignerOpts := []depsync.AssignerOption{
			depsync.WithAssignerLogger(logger.With("component", "assigner")),
		}
		if *flADebug {
			assignerOpts = append(assignerOpts, depsync.WithDebug())
		}
		assigner := depsync.NewAssigner(
			client,
			name,
			storage,
			assignerOpts...,
		)

		// create the callback (that calls the assigner and webhook)
		callback := func(ctx context.Context, isFetch bool, resp *godep.DeviceResponse) error {
			go func() {
				err := assigner.ProcessDeviceResponse(ctx, resp)
				if err != nil {
					logger.Info("msg", "assigner process device response", "err", err)
				}
			}()
			if webhook != nil {
				go func() {
					err := webhook.CallWebhook(ctx, name, isFetch, resp)
					if err != nil {
						logger.Info("msg", "calling webhook", "err", err)
					}
				}()
			}
			return nil
		}

		syncNow := make(chan struct{})
		registerSyncNow(syncNow)

		// create the syncer
		syncerOpts := []depsync.SyncerOption{
			depsync.WithLogger(logger.With("component", "syncer")),
			depsync.WithSyncNow(syncNow),
			depsync.WithCallback(callback),
		}
		if *flDur > 0 {
			syncerOpts = append(syncerOpts, depsync.WithDuration(time.Duration(*flDur)*time.Second)) //nolint:gosec // ignore G115
		}
		if *flLimit > 0 {
			syncerOpts = append(syncerOpts, depsync.WithLimit(*flLimit))
		}
		syncer := depsync.NewSyncer(
			client,
			name,
			storage,
			syncerOpts...,
		)

		// start the syncer
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer closeSyncNow(syncNow)
			err = syncer.Run(ctx)
			if err != nil {
				logger.Info("msg", "syncer run", "err", err)
			}
		}()
	}

	wg.Wait()
}
