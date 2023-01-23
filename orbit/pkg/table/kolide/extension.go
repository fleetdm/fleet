package osquery

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/backoff"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/kolide/kit/version"
	"github.com/kolide/launcher/pkg/service"
	"github.com/mixer/clock"
	"github.com/osquery/osquery-go/plugin/distributed"
	"github.com/osquery/osquery-go/plugin/logger"
	"github.com/pkg/errors"

	"go.etcd.io/bbolt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Extension is the implementation of the osquery extension
// methods. It acts as a communication intermediary between osquery
// and servers -- It provides a grpc and jsonrpc interface for
// osquery. It does not provide any tables.
type Extension struct {
	NodeKey       string
	Opts          ExtensionOpts
	db            *bbolt.DB
	serviceClient service.KolideService
	enrollMutex   sync.Mutex
	done          chan struct{}
	wg            sync.WaitGroup
	logger        log.Logger

	osqueryClient Querier
	initialRunner *initialRunner
}

// SetQuerier sets an osquery client on the extension, allowing
// the extension to query the running osqueryd instance.
func (e *Extension) SetQuerier(client Querier) {
	e.osqueryClient = client
	if e.initialRunner != nil {
		e.initialRunner.client = client
	}
}

// Querier allows querying osquery.
type Querier interface {
	Query(sql string) ([]map[string]string, error)
}

const (
	// Bucket name to use for launcher configuration.
	configBucket         = "config"
	initialResultsBucket = "initial_results"
	// Bucket name to use for buffered status logs.
	statusLogsBucket = "status_logs"
	// Bucket name to use for buffered result logs.
	resultLogsBucket = "result_logs"

	// the bucket which we push values into from server-backed tables
	ServerProvidedDataBucket = "server_provided_data"

	// DB key for UUID
	uuidKey = "uuid"
	// DB key for node key
	nodeKeyKey = "nodeKey"
	// DB key for last retrieved config
	configKey = "config"
	// DB keys for the rsa keys
	privateKeyKey     = "privateKey"
	publicKeyKey      = "publicKey"
	keyFingerprintKey = "keyFingerprint"

	// Default maximum number of bytes per batch (used if not specified in
	// options). This 3MB limit is chosen based on the default grpc-go
	// limit specified in https://github.com/grpc/grpc-go/blob/master/server.go#L51
	// which is 4MB. We use 3MB to be conservative.
	defaultMaxBytesPerBatch = 3 << 20
	// Default logging interval (used if not specified in
	// options)
	defaultLoggingInterval = 60 * time.Second
	// Default maximum number of logs to buffer before purging oldest logs
	// (applies per log type).
	defaultMaxBufferedLogs = 500000
)

// ExtensionOpts is options to be passed in NewExtension
type ExtensionOpts struct {
	// EnrollSecret is the (mandatory) enroll secret used for
	// enrolling with the server.
	EnrollSecret string
	// MaxBytesPerBatch is the maximum number of bytes that should be sent in
	// one batch logging request. Any log larger than this will be dropped.
	MaxBytesPerBatch int
	// LoggingInterval is the interval at which logs should be flushed to
	// the server.
	LoggingInterval time.Duration
	// Clock is the clock that should be used for time based operations. By
	// default it will be a normal realtime clock, but a mock clock can be
	// passed with clock.NewMockClock() for testing purposes.
	Clock clock.Clock
	// Logger is the logger that the extension should use. This is for
	// logging about the launcher, and not for logging osquery results.
	Logger log.Logger
	// MaxBufferedLogs is the maximum number of logs to buffer before
	// purging oldest logs (applies per log type).
	MaxBufferedLogs int
	// RunDifferentialQueriesImmediately allows the client to execute a new query the first time it sees it,
	// bypassing the scheduler.
	RunDifferentialQueriesImmediately bool
}

// NewExtension creates a new Extension from the provided service.KolideService
// implementation. The background routines should be started by calling
// Start().
func NewExtension(client service.KolideService, db *bbolt.DB, opts ExtensionOpts) (*Extension, error) {
	// bucketNames contains the names of buckets that should be created when the
	// extension opens the DB. It should be treated as a constant.
	bucketNames := []string{configBucket, statusLogsBucket, resultLogsBucket, initialResultsBucket, ServerProvidedDataBucket}

	if opts.EnrollSecret == "" {
		return nil, errors.New("empty enroll secret")
	}

	if opts.MaxBytesPerBatch == 0 {
		opts.MaxBytesPerBatch = defaultMaxBytesPerBatch
	}

	if opts.LoggingInterval == 0 {
		opts.LoggingInterval = defaultLoggingInterval
	}

	if opts.Clock == nil {
		opts.Clock = clock.DefaultClock{}
	}

	if opts.Logger == nil {
		// Nop logger
		opts.Logger = log.NewNopLogger()
	}

	if opts.MaxBufferedLogs == 0 {
		opts.MaxBufferedLogs = defaultMaxBufferedLogs
	}

	// Create Bolt buckets as necessary
	err := db.Update(func(tx *bbolt.Tx) error {
		for _, name := range bucketNames {
			_, err := tx.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("creating DB buckets: %w", err)
	}

	if err := SetupLauncherKeys(db); err != nil {
		return nil, fmt.Errorf("setting up initial launcher keys: %w", err)
	}

	identifier, err := IdentifierFromDB(db)
	if err != nil {
		return nil, fmt.Errorf("get host identifier from db when creating new extension: %w", err)
	}

	nodekey, err := NodeKeyFromDB(db)
	if err != nil {
		level.Debug(opts.Logger).Log("msg", "NewExtension got error reading nodekey. Ignoring", "err", err)
		return nil, fmt.Errorf("reading nodekey from db: %w", err)
	} else if nodekey == "" {
		level.Debug(opts.Logger).Log("msg", "NewExtension did not find a nodekey. Likely first enroll")
	} else {
		level.Debug(opts.Logger).Log("msg", "NewExtension found existing nodekey")
	}

	initialRunner := &initialRunner{
		logger:     opts.Logger,
		identifier: identifier,
		db:         db,
		enabled:    opts.RunDifferentialQueriesImmediately,
	}

	return &Extension{
		logger:        opts.Logger,
		serviceClient: client,
		db:            db,
		NodeKey:       nodekey,
		Opts:          opts,
		done:          make(chan struct{}),
		initialRunner: initialRunner,
	}, nil
}

// Start begins the goroutines responsible for background processing (currently
// just the log buffer flushing routine). It should be shut down by calling the
// Shutdown() method.
func (e *Extension) Start() {
	e.wg.Add(1)
	go e.writeLogsLoopRunner()
}

// Shutdown should be called to cleanup the resources and goroutines associated
// with this extension.
func (e *Extension) Shutdown() {
	close(e.done)
	e.wg.Wait()
}

// getHostIdentifier returns the UUID identifier associated with this host. If
// there is an existing identifier, that should be returned. If not, the
// identifier should be randomly generated and persisted.
func (e *Extension) getHostIdentifier() (string, error) {
	return IdentifierFromDB(e.db)
}

func SetupLauncherKeys(db *bbolt.DB) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(configBucket))
		if err != nil {
			return fmt.Errorf("creating bucket: %w", err)
		}

		// This only checks the private key, but it should possibly check all the values we're setting.
		if bucket.Get([]byte(privateKeyKey)) != nil {
			return nil
		}

		key, err := rsaRandomKey()
		if err != nil {
			return fmt.Errorf("generating key: %w", err)
		}

		fingerprint, err := rsaFingerprint(key)
		if err != nil {
			return fmt.Errorf("generating fingerprint: %w", err)
		}

		var pub bytes.Buffer
		if err := RsaPrivateKeyToPem(key, &pub); err != nil {
			return fmt.Errorf("marshalling pub: %w", err)
		}

		keyDer, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return fmt.Errorf("marshalling key: %w", err)
		}

		if err := bucket.Put([]byte(privateKeyKey), keyDer); err != nil {
			return fmt.Errorf("storing key: %w", err)
		}

		if err := bucket.Put([]byte(publicKeyKey), pub.Bytes()); err != nil {
			return fmt.Errorf("storing public key: %w", err)
		}

		if err := bucket.Put([]byte(keyFingerprintKey), []byte(fingerprint)); err != nil {
			return fmt.Errorf("storing fingerprint: %w", err)
		}

		return nil
	})

	return err
}

// PrivateKeyFromDB returns the private launcher key. This is used to authenticate various launcher communications.
func PrivateKeyFromDB(db *bbolt.DB) (*rsa.PrivateKey, error) {
	var privateKey []byte

	if err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(configBucket))
		privateKey = b.Get([]byte(privateKeyKey))
		return nil
	}); err != nil {
		return nil, fmt.Errorf("error reading private key info from db: %w", err)
	}

	key, err := x509.ParsePKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %w", err)
	}

	rsakey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("Private key is not an rsa key")
	}

	return rsakey, nil
}

// PublicKeyFromDB returns the public portions of the launcher key. This is exposed in various launcher info structures.
func PublicKeyFromDB(db *bbolt.DB) (string, string, error) {
	var publicKey []byte
	var fingerprint []byte

	if err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(configBucket))
		publicKey = b.Get([]byte(publicKeyKey))
		fingerprint = b.Get([]byte(keyFingerprintKey))
		return nil
	}); err != nil {
		return "", "", fmt.Errorf("error reading public key info from db: %w", err)
	}

	return string(publicKey), string(fingerprint), nil
}

// IdentifierFromDB returns the built-in launcher identifier from the config bucket.
// The function is exported to allow for building the kolide_launcher_info table.
func IdentifierFromDB(db *bbolt.DB) (string, error) {
	var identifier string
	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(configBucket))
		uuidBytes := b.Get([]byte(uuidKey))
		gotID, err := uuid.ParseBytes(uuidBytes)

		// Use existing UUID
		if err == nil {
			identifier = gotID.String()
			return nil
		}

		// Generate new (random) UUID
		gotID, err = uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("generating new UUID: %w", err)
		}
		identifier = gotID.String()

		// Save new UUID
		err = b.Put([]byte(uuidKey), []byte(identifier))
		if err != nil {
			return fmt.Errorf("saving new UUID: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return identifier, nil
}

// NodeKeyFromDB returns the device node key from a local bolt DB
func NodeKeyFromDB(db *bbolt.DB) (string, error) {
	if db == nil {
		return "", errors.New("received a nil db")
	}

	var key []byte
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(configBucket))
		key = b.Get([]byte(nodeKeyKey))
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error reading node key from db: %w", err)
	}
	if key != nil {
		return string(key), nil
	} else {
		return "", nil
	}
}

// ConfigFromDB returns the device config from a local bolt DB
func ConfigFromDB(db *bbolt.DB) (string, error) {
	if db == nil {
		return "", errors.New("received a nil db")
	}

	var key []byte
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(configBucket))
		key = b.Get([]byte(configKey))
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error reading config from db: %w", err)
	}
	if key != nil {
		return string(key), nil
	} else {
		return "", nil
	}
}

func isNodeInvalidErr(err error) bool {
	err = errors.Cause(err)
	if se, ok := err.(interface{ GRPCStatus() *status.Status }); ok {
		return se.GRPCStatus().Code() == codes.Unauthenticated
	} else {
		return false
	}
}

// Enroll will attempt to enroll the host using the provided enroll secret for
// identification. If the host is already enrolled, the existing node key will
// be returned. To force re-enrollment, use RequireReenroll.
func (e *Extension) Enroll(ctx context.Context) (string, bool, error) {
	logger := log.With(e.logger, "method", "enroll")

	level.Debug(logger).Log("msg", "starting enrollment")

	// Only one thread should ever be allowed to attempt enrollment at the
	// same time.
	e.enrollMutex.Lock()
	defer e.enrollMutex.Unlock()

	// If we already have a successful enrollment (perhaps from another
	// thread), no need to do anything else.
	if e.NodeKey != "" {
		level.Debug(logger).Log("msg", "node key exists, skipping")
		return e.NodeKey, false, nil
	}

	// Look up a node key cached in the local store
	key, err := NodeKeyFromDB(e.db)
	if err != nil {
		return "", false, fmt.Errorf("error reading node key from db: %w", err)
	}

	if key != "" {
		e.NodeKey = key
		return e.NodeKey, false, nil
	}

	identifier, err := e.getHostIdentifier()
	if err != nil {
		return "", true, fmt.Errorf("generating UUID: %w", err)
	}

	// We've seen this fail, so add some retry logic.
	var enrollDetails service.EnrollmentDetails
	if err := backoff.WaitFor(func() error {
		enrollDetails, err = getEnrollDetails(e.osqueryClient)
		return err
	}, 2*time.Minute, 10*time.Second); err != nil {
		if os.Getenv("LAUNCHER_DEBUG_ENROLL_DETAILS_REQUIRED") == "true" {
			return "", true, fmt.Errorf("query enrollment details: %w", err)
		}

		level.Info(logger).Log("msg", "Failed to get enrollment details (even with retries). Moving on", "err", err)
	}

	// If no cached node key, enroll for new node key
	// note that we set invalid two ways. Via the return, _or_ via isNodeInvaliderr
	keyString, invalid, err := e.serviceClient.RequestEnrollment(ctx, e.Opts.EnrollSecret, identifier, enrollDetails)
	if isNodeInvalidErr(err) {
		invalid = true
	} else if err != nil {
		return "", true, fmt.Errorf("transport error in enrollment: %w", err)
	}
	if invalid {
		if err == nil {
			err = errors.New("no further error")
		}
		return "", true, fmt.Errorf("enrollment invalid: %w", err)
	}

	// Save newly acquired node key if successful
	err = e.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(configBucket))
		return b.Put([]byte(nodeKeyKey), []byte(keyString))
	})
	if err != nil {
		return "", true, fmt.Errorf("saving node key: %w", err)
	}

	e.NodeKey = keyString
	return e.NodeKey, false, nil
}

// RequireReenroll clears the existing node key information, ensuring that the
// next call to Enroll will cause the enrollment process to take place.
func (e *Extension) RequireReenroll(ctx context.Context) {
	e.enrollMutex.Lock()
	defer e.enrollMutex.Unlock()
	// Clear the node key such that reenrollment is required.
	e.NodeKey = ""
	e.db.Update(func(tx *bbolt.Tx) error {
		tx.Bucket([]byte(configBucket)).Delete([]byte(nodeKeyKey))
		return nil
	})
}

// GenerateConfigs will request the osquery configuration from the server. If
// retrieving the configuration from the server fails, the locally stored
// configuration will be returned. If that fails, this method will return an
// error.
func (e *Extension) GenerateConfigs(ctx context.Context) (map[string]string, error) {
	config, err := e.generateConfigsWithReenroll(ctx, true)
	if err != nil {
		level.Debug(e.logger).Log(
			"msg", "generating configs with reenroll failed",
			"err", err,
		)
		// Try to use cached config
		var confBytes []byte
		e.db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(configBucket))
			confBytes = b.Get([]byte(configKey))
			return nil
		})

		if len(confBytes) == 0 {
			return nil, fmt.Errorf("loading config failed, no cached config: %w", err)
		}
		config = string(confBytes)
	} else {
		// Store good config
		e.db.Update(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(configBucket))
			return b.Put([]byte(configKey), []byte(config))
		})
		// TODO log or record metrics when caching config fails? We
		// would probably like to return the config and not an error in
		// this case.
	}

	return map[string]string{"config": config}, nil
}

// TODO: https://github.com/kolide/launcher/issues/366
var reenrollmentInvalidErr = errors.New("enrollment invalid, reenrollment invalid")

// Helper to allow for a single attempt at re-enrollment
func (e *Extension) generateConfigsWithReenroll(ctx context.Context, reenroll bool) (string, error) {
	config, invalid, err := e.serviceClient.RequestConfig(ctx, e.NodeKey)
	if isNodeInvalidErr(err) {
		invalid = true
	} else if err != nil {
		return "", fmt.Errorf("transport error retrieving config: %w", err)
	}

	if invalid {
		if err == nil {
			err = errors.New("no further error")
		}

		if !reenroll {
			return "", fmt.Errorf("enrollment invalid, reenroll disabled: %w", err)
		}

		e.RequireReenroll(ctx)
		_, invalid, err := e.Enroll(ctx)
		if err != nil {
			return "", fmt.Errorf("enrollment invalid, reenrollment errored: %w", err)
		}
		if invalid {
			return "", reenrollmentInvalidErr
		}

		// Don't attempt reenroll after first attempt
		return e.generateConfigsWithReenroll(ctx, false)
	}

	if err := e.initialRunner.Execute(config, e.writeLogsWithReenroll); err != nil {
		return "", fmt.Errorf("initial run results: %w", err)
	}

	return config, nil
}

// byteKeyFromUint64 turns a uint64 (generated by Bolt's NextSequence) into a
// sortable byte slice to use as a key.
func byteKeyFromUint64(k uint64) []byte {
	// Adapted from Bolt docs
	// 8 bytes in a uint64
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, k)
	return b
}

// uint64FromByteKey turns a byte slice (retrieved as the key from Bolt) into a
// uint64
func uint64FromByteKey(k []byte) uint64 {
	return binary.BigEndian.Uint64(k)
}

// bucketNameFromLogType returns the Bolt bucket name that stores logs of the
// provided type.
func bucketNameFromLogType(typ logger.LogType) (string, error) {
	switch typ {
	case logger.LogTypeString, logger.LogTypeSnapshot:
		return resultLogsBucket, nil
	case logger.LogTypeStatus:
		return statusLogsBucket, nil
	default:
		return "", fmt.Errorf("unknown log type: %v", typ)

	}
}

// writeAndPurgeLogs flushes the log buffers, writing up to
// Opts.MaxBytesPerBatch bytes in one run. If the logs write successfully, they
// will be deleted from the buffer. After writing (whether success or failure),
// logs over the maximum count will be purged to avoid unbounded growth of the
// buffers.
func (e *Extension) writeAndPurgeLogs() {
	for _, typ := range []logger.LogType{logger.LogTypeStatus, logger.LogTypeString} {
		// Write logs
		err := e.writeBufferedLogsForType(typ)
		if err != nil {
			level.Info(e.Opts.Logger).Log(
				"err", fmt.Errorf("sending %v logs: %w", typ, err),
			)
		}

		// Purge overflow
		err = e.purgeBufferedLogsForType(typ)
		if err != nil {
			level.Info(e.Opts.Logger).Log(
				"err", fmt.Errorf("purging %v logs: %w", typ, err),
			)
		}
	}
}

func (e *Extension) writeLogsLoopRunner() {
	defer e.wg.Done()
	ticker := e.Opts.Clock.NewTicker(e.Opts.LoggingInterval)
	defer ticker.Stop()
	for {
		e.writeAndPurgeLogs()

		// select to either exit or write another batch of logs
		select {
		case <-e.done:
			return
		case <-ticker.Chan():
			// Resume loop
		}
	}
}

// numberOfBufferedLogs returns the number of logs buffered for a given type.
func (e *Extension) numberOfBufferedLogs(typ logger.LogType) (int, error) {
	bucketName, err := bucketNameFromLogType(typ)
	if err != nil {
		return 0, err
	}

	var count int
	err = e.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		count = b.Stats().KeyN
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("counting buffered logs: %w", err)
	}

	return count, nil
}

// writeBufferedLogs flushes the log buffers, writing up to
// Opts.MaxBytesPerBatch bytes worth of logs in one run. If the logs write
// successfully, they will be deleted from the buffer.
func (e *Extension) writeBufferedLogsForType(typ logger.LogType) error {
	bucketName, err := bucketNameFromLogType(typ)
	if err != nil {
		return err
	}

	// Collect up logs to be sent
	var logs []string
	var logIDs [][]byte
	err = e.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		c := b.Cursor()
		k, v := c.First()
		for totalBytes := 0; k != nil; {
			// A somewhat cumbersome if block...
			//
			// 1. If the log is too big, skip it and mark for deletion.
			// 2. If the buffer would be too big with the log, break for
			// 3. Else append it
			//
			// Note that (1) must come first, otherwise (2) will always trigger.
			if len(v) > e.Opts.MaxBytesPerBatch {
				// Discard logs that are too big
				logheadSize := minInt(len(v), 100)
				level.Info(e.Opts.Logger).Log(
					"msg", "dropped log",
					"logID", k,
					"size", len(v),
					"limit", e.Opts.MaxBytesPerBatch,
					"loghead", string(v)[0:logheadSize],
				)
			} else if totalBytes+len(v) > e.Opts.MaxBytesPerBatch {
				// Buffer is filled. Break the loop and come back later.
				break
			} else {
				logs = append(logs, string(v))
				totalBytes += len(v)
			}

			// Note the logID for deletion. We do this by
			// making a copy of k. It is retained in
			// logIDs after the transaction is closed,
			// when the goroutine ticks it zeroes out some
			// of the IDs to delete below, causing logs to
			// remain in the buffer and be sent again to
			// the server.
			logID := make([]byte, len(k))
			copy(logID, k)
			logIDs = append(logIDs, logID)

			k, v = c.Next()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reading buffered logs: %w", err)
	}

	if len(logs) == 0 {
		// Nothing to send
		return nil
	}

	err = e.writeLogsWithReenroll(context.Background(), typ, logs, true)
	if err != nil {
		return fmt.Errorf("writing logs: %w", err)
	}

	// Delete logs that were successfully sent
	err = e.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		for _, k := range logIDs {
			b.Delete(k)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("deleting sent logs: %w", err)
	}

	return nil
}

// Helper to allow for a single attempt at re-enrollment
func (e *Extension) writeLogsWithReenroll(ctx context.Context, typ logger.LogType, logs []string, reenroll bool) error {
	_, _, invalid, err := e.serviceClient.PublishLogs(ctx, e.NodeKey, typ, logs)
	if isNodeInvalidErr(err) {
		invalid = true
	} else if err != nil {
		return fmt.Errorf("transport error sending logs: %w", err)
	}

	if invalid {
		if !reenroll {
			return errors.New("enrollment invalid, reenroll disabled")
		}

		e.RequireReenroll(ctx)
		_, invalid, err := e.Enroll(ctx)
		if err != nil {
			return fmt.Errorf("enrollment invalid, reenrollment errored: %w", err)
		}
		if invalid {
			return errors.New("enrollment invalid, reenrollment invalid")
		}

		// Don't attempt reenroll after first attempt
		return e.writeLogsWithReenroll(ctx, typ, logs, false)
	}

	return nil
}

// purgeBufferedLogsForType flushes the log buffers for the provided type,
// ensuring that at most Opts.MaxBufferedLogs logs remain.
func (e *Extension) purgeBufferedLogsForType(typ logger.LogType) error {
	bucketName, err := bucketNameFromLogType(typ)
	if err != nil {
		return err
	}
	err = e.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		logCount := b.Stats().KeyN
		deleteCount := logCount - e.Opts.MaxBufferedLogs

		if deleteCount <= 0 {
			// Limit not exceeded
			return nil
		}

		level.Info(e.Opts.Logger).Log(
			"msg", "Buffered logs limit exceeded. Purging excess.",
			"limit", e.Opts.MaxBufferedLogs,
			"purge_count", deleteCount,
		)

		c := b.Cursor()
		k, _ := c.First()
		for total := 0; k != nil && total < deleteCount; total++ {
			c.Delete() // Note: This advances the cursor
			k, _ = c.First()
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("deleting overflowed logs: %w", err)
	}
	return nil
}

// LogString will buffer logs from osquery into the local BoltDB store. No
// immediate action is taken to push the logs to the server (that is handled by
// the log publishing thread).
func (e *Extension) LogString(ctx context.Context, typ logger.LogType, logText string) error {
	if typ == logger.LogTypeInit {
		// osquery seems to send an empty init log whenever it starts
		// up. Ignore this log without generating an error.
		return nil
	}

	bucketName, err := bucketNameFromLogType(typ)
	if err != nil {
		level.Info(e.Opts.Logger).Log(
			"msg", "Received unknown log type",
			"log_type", typ,
		)
		return fmt.Errorf("unknown log type: %w", err)
	}

	// Buffer the log for sending later in a batch
	err = e.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		// Log keys are generated with the auto-incrementing sequence
		// number provided by BoltDB. These must be converted to []byte
		// (which we do with byteKeyFromUint64 function).
		key, err := b.NextSequence()
		if err != nil {
			return fmt.Errorf("generating key: %w", err)
		}

		return b.Put(byteKeyFromUint64(key), []byte(logText))
	})

	if err != nil {
		return fmt.Errorf("buffering log: %w", err)
	}

	return nil
}

// GetQueries will request the distributed queries to execute from the server.
func (e *Extension) GetQueries(ctx context.Context) (*distributed.GetQueriesResult, error) {
	return e.getQueriesWithReenroll(ctx, true)
}

// Helper to allow for a single attempt at re-enrollment
func (e *Extension) getQueriesWithReenroll(ctx context.Context, reenroll bool) (*distributed.GetQueriesResult, error) {
	// Note that we set invalid two ways -- in the return, and via isNodeinvaliderr
	queries, invalid, err := e.serviceClient.RequestQueries(ctx, e.NodeKey)
	if isNodeInvalidErr(err) {
		invalid = true
	} else if err != nil {
		return nil, fmt.Errorf("transport error getting queries: %w", err)
	}

	if invalid {
		if err == nil {
			err = errors.New("no further error")
		}

		if !reenroll {
			return nil, fmt.Errorf("enrollment invalid, reenroll disabled: %w", err)
		}

		e.RequireReenroll(ctx)
		_, invalid, err := e.Enroll(ctx)
		if err != nil {
			return nil, fmt.Errorf("enrollment invalid, reenrollment errored: %w", err)
		}
		if invalid {
			return nil, errors.New("enrollment invalid, reenrollment invalid")
		}

		// Don't attempt reenroll after first attempt
		return e.getQueriesWithReenroll(ctx, false)
	}

	return queries, nil
}

// WriteResults will publish results of the executed distributed queries back
// to the server.
func (e *Extension) WriteResults(ctx context.Context, results []distributed.Result) error {
	return e.writeResultsWithReenroll(ctx, results, true)
}

// Helper to allow for a single attempt at re-enrollment
func (e *Extension) writeResultsWithReenroll(ctx context.Context, results []distributed.Result, reenroll bool) error {
	_, _, invalid, err := e.serviceClient.PublishResults(ctx, e.NodeKey, results)
	if isNodeInvalidErr(err) {
		invalid = true
	} else if err != nil {
		return fmt.Errorf("transport error writing results: %w", err)
	}

	if invalid {
		if !reenroll {
			return errors.New("enrollment invalid, reenroll disabled")
		}

		e.RequireReenroll(ctx)
		_, invalid, err := e.Enroll(ctx)
		if err != nil {
			return fmt.Errorf("enrollment invalid, reenrollment errored: %w", err)
		}
		if invalid {
			return errors.New("enrollment invalid, reenrollment invalid")
		}

		// Don't attempt reenroll after first attempt
		return e.writeResultsWithReenroll(ctx, results, false)
	}

	return nil
}

func getEnrollDetails(client Querier) (service.EnrollmentDetails, error) {
	var details service.EnrollmentDetails

	// To facilitate manual testing around missing enrollment details,
	// there is a environmental variable to trigger the failure condition
	if os.Getenv("LAUNCHER_DEBUG_ENROLL_DETAILS_ERROR") == "true" {
		return details, errors.New("Skipping enrollment details")
	}

	// This condition is indicative of a misordering (or race) in
	// startup. Enrollment has started before `SetQuerier` has
	// been called.
	if client == nil {
		return details, errors.New("no querier")
	}

	query := `
	SELECT
		osquery_info.version as osquery_version,
		os_version.build as os_build,
		os_version.name as os_name,
		os_version.platform as os_platform,
		os_version.platform_like as os_platform_like,
		os_version.version as os_version,
		system_info.hardware_model,
		system_info.hardware_serial,
		system_info.hardware_vendor,
		system_info.hostname,
		system_info.uuid as hardware_uuid
	FROM
		os_version,
		system_info,
		osquery_info;
`
	resp, err := client.Query(query)
	if err != nil {
		return details, fmt.Errorf("query enrollment details: %w", err)
	}

	if len(resp) < 1 {
		return details, errors.New("expected at least one row from the enrollment details query")
	}

	if val, ok := resp[0]["os_version"]; ok {
		details.OSVersion = val
	}
	if val, ok := resp[0]["os_build"]; ok {
		details.OSBuildID = val
	}
	if val, ok := resp[0]["os_name"]; ok {
		details.OSName = val
	}
	if val, ok := resp[0]["os_platform"]; ok {
		details.OSPlatform = val
	}
	if val, ok := resp[0]["os_platform_like"]; ok {
		details.OSPlatformLike = val
	}
	if val, ok := resp[0]["osquery_version"]; ok {
		details.OsqueryVersion = val
	}
	if val, ok := resp[0]["hardware_model"]; ok {
		details.HardwareModel = val
	}
	details.HardwareSerial = serialForRow(resp[0])
	if val, ok := resp[0]["hardware_vendor"]; ok {
		details.HardwareVendor = val
	}
	if val, ok := resp[0]["hostname"]; ok {
		details.Hostname = val
	}

	if val, ok := resp[0]["hardware_uuid"]; ok {
		details.HardwareUUID = val
	}

	// This runs before the extensions are registered. These mirror the
	// underlying tables.
	details.LauncherVersion = version.Version().Version
	details.GOOS = runtime.GOOS
	details.GOARCH = runtime.GOARCH

	return details, nil
}

type initialRunner struct {
	logger     log.Logger
	enabled    bool
	identifier string
	client     Querier
	db         *bbolt.DB
}

func (i *initialRunner) Execute(configBlob string, writeFn func(ctx context.Context, l logger.LogType, results []string, reeenroll bool) error) error {
	var config OsqueryConfig
	if err := json.Unmarshal([]byte(configBlob), &config); err != nil {
		return fmt.Errorf("unmarshal osquery config blob: %w", err)
	}

	var allQueries []string
	for packName, pack := range config.Packs {
		// only run queries from kolide packs
		if !strings.Contains(packName, "_kolide_") {
			continue
		}

		// Run all the queries, snapshot and differential
		for query := range pack.Queries {
			queryName := fmt.Sprintf("pack:%s:%s", packName, query)
			allQueries = append(allQueries, queryName)
		}
	}

	toRun, err := i.queriesToRun(allQueries)
	if err != nil {
		return fmt.Errorf("checking if query should run: %w", err)
	}

	var initialRunResults []OsqueryResultLog
	for packName, pack := range config.Packs {
		if !i.enabled { // only execute them when the plugin is enabled.
			break
		}
		for query, queryContent := range pack.Queries {
			queryName := fmt.Sprintf("pack:%s:%s", packName, query)
			if _, ok := toRun[queryName]; !ok {
				continue
			}
			resp, err := i.client.Query(queryContent.Query)
			// returning here causes the rest of the queries not to run
			// this is a bummer because often configs have queries with bad syntax/tables that do not exist.
			// log the error and move on.
			// using debug to not fill disks. the worst that will happen is that the result will come in later.
			level.Debug(i.logger).Log(
				"msg", "querying for initial results",
				"query_name", queryName,
				"err", err,
				"results", len(resp),
			)
			if err != nil || len(resp) == 0 {
				continue
			}

			initialRunResults = append(initialRunResults, OsqueryResultLog{
				Name:           queryName,
				HostIdentifier: i.identifier,
				UnixTime:       int(time.Now().UTC().Unix()),
				DiffResults:    &DiffResults{Added: resp},
			})
		}
	}

	cctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, result := range initialRunResults {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(result); err != nil {
			return fmt.Errorf("encoding initial run result: %w", err)
		}
		if err := writeFn(cctx, logger.LogTypeString, []string{buf.String()}, true); err != nil {
			level.Debug(i.logger).Log(
				"msg", "writing initial result log to server",
				"query_name", result.Name,
				"err", err,
			)
			continue
		}
	}

	// note: caching would happen always on first use, even if the runner is not enabled.
	// This avoids the problem of queries not being known even though they've been in the config for a long time.
	if err := i.cacheRanQueries(toRun); err != nil {
		return err
	}

	return nil
}

func (i *initialRunner) queriesToRun(allFromConfig []string) (map[string]struct{}, error) {
	known := make(map[string]struct{})
	err := i.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(initialResultsBucket))
		for _, q := range allFromConfig {
			knownQuery := b.Get([]byte(q))
			if knownQuery != nil {
				continue
			}
			known[q] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("check bolt for queries to run: %w", err)
	}

	return known, nil
}

func (i *initialRunner) cacheRanQueries(known map[string]struct{}) error {
	err := i.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(initialResultsBucket))
		for q := range known {
			if err := b.Put([]byte(q), []byte(q)); err != nil {
				return fmt.Errorf("cache initial result query %q: %w", q, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("cache initial result queries: %w", err)
	}

	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
