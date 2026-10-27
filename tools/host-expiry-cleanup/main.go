package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/WatchBeam/clock"
	activitybootstrap "github.com/fleetdm/fleet/v4/server/activity/bootstrap"
	"github.com/fleetdm/fleet/v4/server/config"
	fleetmysql "github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	batchSize       = 65_000
	leaderLockTTL   = 2 * time.Minute
	leaderLockRenew = 30 * time.Second
	leaderLockWait  = 10 * time.Minute
	cronYield       = time.Minute
	lockRetryDelay  = 5 * time.Second
)

type seenTimeBucket struct {
	start time.Time
	end   time.Time
}

type workerState struct {
	bucketIndex int
	done        bool
	deleted     int64
}

type cleanupBatchFunc func(context.Context, int) ([]fleet.DeletedHostDetails, error)

func main() {
	var apply bool
	var estimate bool
	var maxRounds int
	var workerCount int
	var bucketDays int
	var serialOnly bool
	root := &cobra.Command{Use: "host-expiry-cleanup", Args: cobra.NoArgs}
	configPath := root.PersistentFlags().StringP("config", "c", "", "Fleet config file on this machine")
	root.Flags().BoolVar(&apply, "apply", false, "Delete expired hosts; without this flag only print the planned time buckets")
	root.Flags().BoolVar(&estimate, "estimate", false, "Use MySQL optimizer metadata to estimate old OSQuery-seen rows; do not delete hosts")
	root.Flags().BoolVar(&serialOnly, "serial-only", false, "Skip seen-time buckets and run the unscoped cleanup pass only")
	root.Flags().IntVar(&maxRounds, "max-rounds", 0, "Stop after this many cleanup rounds (0 means no limit)")
	root.Flags().IntVar(&workerCount, "workers", 4, "Number of concurrent cleanup workers")
	root.Flags().IntVar(&bucketDays, "bucket-days", 30, "Number of days in each non-overlapping seen-time bucket")
	manager := config.NewManager(root)
	root.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name != "config" {
			flag.Hidden = true
		}
	})
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		if *configPath == "" {
			return errors.New("--config is required")
		}
		if maxRounds < 0 {
			return errors.New("--max-rounds must be non-negative")
		}
		if workerCount <= 0 {
			return errors.New("--workers must be positive")
		}
		if bucketDays <= 0 {
			return errors.New("--bucket-days must be positive")
		}
		if estimate && apply {
			return errors.New("--estimate and --apply cannot be used together")
		}
		cfg := manager.LoadConfig()
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		cfg.Mysql.MaxOpenConns, cfg.Mysql.MaxIdleConns = 15, 15
		cfg.MysqlReadReplica.MaxOpenConns, cfg.MysqlReadReplica.MaxIdleConns = 2, 2
		dbOptions := []fleetmysql.DBOption{fleetmysql.Logger(logger)}
		if cfg.MysqlReadReplica.Address != "" {
			dbOptions = append(dbOptions, fleetmysql.Replica(&cfg.MysqlReadReplica))
		}
		dbConns, err := fleetmysql.NewDBConnections(cfg.Mysql, dbOptions...)
		if err != nil {
			return err
		}
		defer dbConns.Primary.Close()
		if dbConns.Replica != dbConns.Primary {
			defer dbConns.Replica.Close()
		}

		ds, err := fleetmysql.NewDatastore(dbConns, cfg.Mysql, clock.C)
		if err != nil {
			return err
		}
		if estimate {
			return estimateExpiredHostSeenRows(cmd.Context(), dbConns.Primary, ds, logger)
		}
		var buckets []seenTimeBucket
		if !serialOnly {
			bucketSize := time.Duration(bucketDays) * 24 * time.Hour
			buckets, err = makeSeenTimeBuckets(cmd.Context(), dbConns.Primary, bucketSize)
			if err != nil {
				return err
			}
			logger.Info("planned host-expiry work", "workers", workerCount, "bucket_days", bucketDays, "buckets", len(buckets), "batch_size_per_worker", batchSize, "deleted_host_activities", "omitted", "activity_webhooks", "omitted")
			for i, bucket := range buckets {
				logger.Info("time bucket", "bucket", i+1, "start", bucket.start.Format(time.RFC3339), "end", bucket.end.Format(time.RFC3339))
			}
		} else {
			logger.Info("planned host-expiry work", "mode", "serial-only", "batch_size", batchSize, "deleted_host_activities", "omitted", "activity_webhooks", "omitted")
		}
		if !apply {
			if serialOnly {
				fmt.Println("Dry run only; no hosts were deleted. Rerun with --apply to execute the unscoped serial cleanup pass.")
			} else {
				fmt.Println("Dry run only; no hosts were deleted. Rerun with --apply to execute these buckets.")
			}
			return nil
		}

		cleanupBatch, err := newCleanupBatch(ds, dbConns, logger)
		if err != nil {
			return err
		}
		owner, err := newLockOwner()
		if err != nil {
			return err
		}

		stop := make(chan struct{})
		var stopOnce sync.Once
		requestStop := func() { stopOnce.Do(func() { close(stop) }) }
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(signalCh)
		go func() {
			<-signalCh
			logger.Info("stop requested; workers will finish their current batch before stopping")
			requestStop()
		}()

		var deleted int64
		var roundsCompleted int
		if !serialOnly {
			deleted, roundsCompleted, err = runBucketWorkers(cmd.Context(), ds, cleanupBatch, buckets, workerCount, owner, logger, stop, requestStop, maxRounds)
			if err != nil {
				return err
			}
			if isStopped(stop) {
				logger.Info("stopped after completing active batches", "deleted", deleted)
				return nil
			}
		}

		// Time buckets cover hosts with OSQuery seen-time rows. Keep Fleet's normal
		// unscoped cleanup as a final serial pass for MDM-only and never-seen hosts.
		logger.Info("starting serial cleanup pass for MDM-only and never-seen hosts")
		for {
			if maxRounds > 0 && roundsCompleted >= maxRounds {
				logger.Info("stopped at requested round limit", "rounds", roundsCompleted, "deleted", deleted)
				return nil
			}
			if isStopped(stop) {
				logger.Info("stopped before the next fallback batch", "deleted", deleted)
				return nil
			}
			var batch []fleet.DeletedHostDetails
			err := runWithLeaderLock(cmd.Context(), ds, owner, stop, logger, func(batchCtx context.Context) error {
				var batchErr error
				batch, batchErr = cleanupBatch(batchCtx, batchSize)
				return batchErr
			})
			roundsCompleted++
			if err != nil {
				if isStopped(stop) {
					return nil
				}
				return err
			}
			if isStopped(stop) {
				logger.Info("stopped before the next fallback batch", "deleted", deleted)
				return nil
			}
			if len(batch) == 0 {
				break
			}
			deleted += int64(len(batch))
			logger.Info("cleaned fallback host batch", "deleted", len(batch), "total_deleted", deleted)
			if sleepOrStop(stop, cronYield) {
				logger.Info("stopped after completing the fallback batch", "deleted", deleted)
				return nil
			}
		}
		logger.Info("host-expiry cleanup complete", "deleted", deleted)
		return nil
	}
	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}

func estimateExpiredHostSeenRows(ctx context.Context, db *sqlx.DB, ds *fleetmysql.Datastore, logger *slog.Logger) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	var teams []*fleet.Team
	if err := sqlx.SelectContext(ctx, db, &teams, `SELECT id, config FROM teams`); err != nil {
		return err
	}
	shortestExpiryWindow := 0
	if appConfig.HostExpirySettings.HostExpiryEnabled {
		shortestExpiryWindow = appConfig.HostExpirySettings.HostExpiryWindow
	}
	for _, team := range teams {
		settings := team.Config.HostExpirySettings
		if settings.HostExpiryEnabled && (shortestExpiryWindow == 0 || settings.HostExpiryWindow < shortestExpiryWindow) {
			shortestExpiryWindow = settings.HostExpiryWindow
		}
	}
	if shortestExpiryWindow <= 0 {
		fmt.Println("Host expiry is disabled for the global configuration and all teams.")
		return nil
	}

	var tableRows sql.NullInt64
	if err := db.GetContext(ctx, &tableRows, `SELECT table_rows FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'host_seen_times'`); err != nil {
		return err
	}
	var histogram sql.NullString
	histogramErr := db.GetContext(ctx, &histogram, `SELECT histogram FROM information_schema.column_statistics WHERE schema_name = DATABASE() AND table_name = 'host_seen_times' AND column_name = 'seen_time'`)
	if histogramErr != nil && !errors.Is(histogramErr, sql.ErrNoRows) {
		return histogramErr
	}
	query := fmt.Sprintf(`EXPLAIN FORMAT=JSON SELECT host_id FROM host_seen_times FORCE INDEX (idx_host_seen_times_seen_time) WHERE seen_time < DATE_SUB(NOW(), INTERVAL %d DAY)`, shortestExpiryWindow)
	var planJSON string
	if err := db.GetContext(ctx, &planJSON, query); err != nil {
		return err
	}
	var plan any
	if err := json.Unmarshal([]byte(planJSON), &plan); err != nil {
		return fmt.Errorf("parse EXPLAIN metadata: %w", err)
	}
	estimatedRows, ok := findRowsExaminedEstimate(plan)
	if !ok {
		return errors.New("EXPLAIN output did not include rows_examined_per_scan")
	}
	logger.Info("expired host metadata estimate", "expiry_window_days", shortestExpiryWindow, "potential_old_osquery_rows", estimatedRows, "host_seen_times_table_rows_estimate", tableRows.Int64, "has_seen_time_histogram", histogramErr == nil && histogram.Valid, "estimate_source", "MySQL EXPLAIN index statistics")
	logger.Info("MySQL EXPLAIN plan for metadata estimate", "plan", planJSON)
	return nil
}

func findRowsExaminedEstimate(value any) (float64, bool) {
	switch v := value.(type) {
	case map[string]any:
		if rows, ok := v["rows_examined_per_scan"].(float64); ok {
			return rows, true
		}
		for _, child := range v {
			if rows, ok := findRowsExaminedEstimate(child); ok {
				return rows, true
			}
		}
	case []any:
		for _, child := range v {
			if rows, ok := findRowsExaminedEstimate(child); ok {
				return rows, true
			}
		}
	}
	return 0, false
}

func makeSeenTimeBuckets(ctx context.Context, db *sqlx.DB, bucketSize time.Duration) ([]seenTimeBucket, error) {
	var minSeen sql.NullTime
	if err := db.GetContext(ctx, &minSeen, `SELECT MIN(seen_time) FROM host_seen_times`); err != nil {
		return nil, err
	}
	if !minSeen.Valid {
		return nil, nil
	}
	end := time.Now().UTC()
	start := minSeen.Time.UTC().Truncate(bucketSize)
	var buckets []seenTimeBucket
	for start.Before(end) {
		next := start.Add(bucketSize)
		if next.After(end) {
			next = end
		}
		buckets = append(buckets, seenTimeBucket{start: start, end: next})
		start = next
	}
	return buckets, nil
}

func newCleanupBatch(ds *fleetmysql.Datastore, dbConns *platform_mysql.DBConnections, logger *slog.Logger) (cleanupBatchFunc, error) {
	// Keep orphaned host/activity links tidy without writing one audit activity per deleted host.
	activitySvc, err := activitybootstrap.New(dbConns, nil, nil, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize host activity cleanup: %w", err)
	}
	return func(ctx context.Context, batchSize int) ([]fleet.DeletedHostDetails, error) {
		hostDetails, err := ds.CleanupExpiredHostsBatch(ctx, batchSize)
		if err != nil || len(hostDetails) == 0 {
			return hostDetails, err
		}
		hostIDs := make([]uint, 0, len(hostDetails))
		for _, host := range hostDetails {
			hostIDs = append(hostIDs, host.ID)
		}
		if err := activitySvc.CleanupHostActivities(ctx, hostIDs); err != nil {
			return nil, fmt.Errorf("cleanup host activities after expired host deletion: %w", err)
		}
		logger.Info("cleaned activity links; skipped deleted-host activities and webhooks", "hosts", len(hostDetails))
		return hostDetails, nil
	}, nil
}

func runBucketWorkers(ctx context.Context, ds fleet.Datastore, cleanupBatch cleanupBatchFunc, buckets []seenTimeBucket, workerCount int, owner string, logger *slog.Logger, stop <-chan struct{}, requestStop func(), maxRounds int) (int64, int, error) {
	if len(buckets) == 0 {
		return 0, 0, nil
	}
	var nextBucket atomic.Int64
	states := make([]workerState, workerCount)
	for i := range states {
		states[i].bucketIndex = -1
	}
	var totalDeleted int64
	var roundsCompleted int
	for {
		if isStopped(stop) {
			return totalDeleted, roundsCompleted, nil
		}
		active := false
		for i := range states {
			if states[i].done {
				continue
			}
			if states[i].bucketIndex < 0 {
				states[i].bucketIndex = int(nextBucket.Add(1) - 1)
				if states[i].bucketIndex >= len(buckets) {
					states[i].done = true
					continue
				}
			}
			active = true
		}
		if !active {
			return totalDeleted, roundsCompleted, nil
		}

		var roundDeleted int64
		err := runWithLeaderLock(ctx, ds, owner, stop, logger, func(roundCtx context.Context) error {
			if isStopped(stop) {
				return nil
			}
			var wg sync.WaitGroup
			workerErrs := make([]error, workerCount)
			workerDeleted := make([]int64, workerCount)
			wg.Add(workerCount)
			for i := range states {
				if states[i].done {
					wg.Done()
					continue
				}
				workerIndex := i
				workerID := workerIndex + 1
				state := &states[workerIndex]
				go func() {
					defer wg.Done()
					for {
						bucketIndex := state.bucketIndex
						bucket := buckets[bucketIndex]
						rangeCtx, err := fleetmysql.WithExpiredHostsSeenTimeRange(roundCtx, bucket.start, bucket.end, workerID)
						if err != nil {
							workerErrs[workerIndex] = err
							return
						}
						batch, err := cleanupBatch(rangeCtx, batchSize)
						if err != nil {
							workerErrs[workerIndex] = err
							return
						}
						if len(batch) > 0 {
							state.deleted += int64(len(batch))
							workerDeleted[workerIndex] = int64(len(batch))
							logger.Info("deleted expired-host batch", "worker", workerID, "bucket", bucketIndex+1, "deleted", len(batch), "bucket_total", state.deleted)
							return
						}
						logger.Info("finished expired-host time bucket", "worker", workerID, "bucket", bucketIndex+1, "start", bucket.start.Format(time.RFC3339), "end", bucket.end.Format(time.RFC3339), "deleted_in_bucket", state.deleted)
						state.bucketIndex = int(nextBucket.Add(1) - 1)
						state.deleted = 0
						if state.bucketIndex >= len(buckets) {
							state.bucketIndex = -1
							state.done = true
							return
						}
					}
				}()
			}
			wg.Wait()
			for _, deleted := range workerDeleted {
				roundDeleted += deleted
			}
			return errors.Join(workerErrs...)
		})
		totalDeleted += roundDeleted
		if err != nil {
			requestStop()
			return totalDeleted, roundsCompleted, err
		}
		roundsCompleted++
		if roundDeleted > 0 {
			logger.Info("completed parallel cleanup round", "deleted", roundDeleted, "total_deleted", totalDeleted)
		}
		if isStopped(stop) {
			return totalDeleted, roundsCompleted, nil
		}
		if maxRounds > 0 && roundsCompleted >= maxRounds {
			logger.Info("stopped at requested round limit", "rounds", roundsCompleted, "deleted", totalDeleted)
			return totalDeleted, roundsCompleted, nil
		}
		if sleepOrStop(stop, cronYield) {
			return totalDeleted, roundsCompleted, nil
		}
	}
}

func runWithLeaderLock(ctx context.Context, ds fleet.Datastore, owner string, stop <-chan struct{}, logger *slog.Logger, work func(context.Context) error) error {
	if err := waitForCronTurn(ctx, ds, owner, stop, logger); err != nil {
		return err
	}
	if isStopped(stop) {
		return ds.Unlock(context.Background(), "leader", owner)
	}

	workCtx, cancelWork := context.WithCancel(ctx)
	stopRenewal := make(chan struct{})
	renewalDone := make(chan struct{})
	renewalErr := make(chan error, 1)
	go func() {
		defer close(renewalDone)
		ticker := time.NewTicker(leaderLockRenew)
		defer ticker.Stop()
		for {
			select {
			case <-stopRenewal:
				return
			case <-ticker.C:
				ok, err := ds.Lock(workCtx, "leader", owner, leaderLockTTL)
				if err != nil {
					select {
					case renewalErr <- fmt.Errorf("renew Fleet cleanup schedule lock: %w", err):
					default:
					}
					cancelWork()
					return
				}
				if !ok {
					select {
					case renewalErr <- errors.New("Fleet cleanup schedule lock ownership was lost"):
					default:
					}
					cancelWork()
					return
				}
			}
		}
	}()
	workErr := work(workCtx)
	close(stopRenewal)
	cancelWork()
	<-renewalDone
	unlockErr := ds.Unlock(context.Background(), "leader", owner)
	select {
	case err := <-renewalErr:
		return errors.Join(workErr, err, unlockErr)
	default:
		return errors.Join(workErr, unlockErr)
	}
}

func waitForCronTurn(ctx context.Context, ds fleet.Datastore, owner string, stop <-chan struct{}, logger *slog.Logger) error {
	waitCtx, cancelWait := context.WithTimeout(ctx, leaderLockWait)
	defer cancelWait()
	for {
		if isStopped(stop) {
			return nil
		}
		lockCtx, cancel := context.WithTimeout(waitCtx, 15*time.Second)
		ok, err := ds.Lock(lockCtx, "leader", owner, leaderLockTTL)
		cancel()
		if err != nil {
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("timed out waiting %s for Fleet cleanup schedule lock", leaderLockWait)
			}
			return fmt.Errorf("acquire Fleet cleanup schedule lock: %w", err)
		}
		if ok {
			return nil
		}
		logger.Info("Fleet cleanup schedule is running; waiting before next worker batch")
		timer := time.NewTimer(lockRetryDelay)
		select {
		case <-stop:
			timer.Stop()
			return nil
		case <-waitCtx.Done():
			timer.Stop()
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("timed out waiting %s for Fleet cleanup schedule lock", leaderLockWait)
			}
			return waitCtx.Err()
		case <-timer.C:
		}
	}
}

func newLockOwner() (string, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(random[:]), nil
}

func sleepOrStop(stop <-chan struct{}, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-stop:
		return true
	case <-timer.C:
		return false
	}
}

func isStopped(stop <-chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}
