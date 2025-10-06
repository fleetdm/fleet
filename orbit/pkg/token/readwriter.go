package token

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type remoteUpdaterFunc func(token string) error

type ReadWriter struct {
	*Reader
	remoteUpdate           remoteUpdaterFunc
	rotationCheckerStarted int
	checkTokenFunc         func(token string) error
	localCheckDuration     time.Duration
	remoteCheckDuration    time.Duration
	rotationStopCh         chan struct{}
}

func NewReadWriter(path string, checkTokenFunc func(token string) error) *ReadWriter {
	return &ReadWriter{
		Reader:              &Reader{Path: path},
		checkTokenFunc:      checkTokenFunc,
		localCheckDuration:  30 * time.Second,
		remoteCheckDuration: 5 * time.Minute,
	}
}

// LoadOrGenerate tries to read a token file from disk,
// and if it doesn't exist, generates a new one.
func (rw *ReadWriter) LoadOrGenerate() error {
	_, err := rw.Read()
	switch {
	case err == nil:
		// ensure the file is readable by other processes, old versions of Orbit
		// used to chmod this file with 0o600
		if err := rw.setChmod(); err != nil {
			return fmt.Errorf("loading token file, chmod %q: %w", rw.Path, err)
		}
	case errors.Is(err, os.ErrNotExist):
		if err := rw.Rotate(); err != nil {
			return fmt.Errorf("rotating token on generation: %w", err)
		}
	default:
		return fmt.Errorf("load identifier file %q: %w", rw.Path, err)
	}
	return nil
}

// Rotate assigns a new value to the token and writes it to disk.
func (rw *ReadWriter) Rotate() error {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("generate identifier: %w", err)
	}

	id := uuid.String()
	attempts := 3
	interval := 5 * time.Second
	err = retry.Do(func() error {
		return rw.Write(id)
	}, retry.WithMaxAttempts(attempts), retry.WithInterval(interval))
	if err != nil {
		return fmt.Errorf("saving token after %d attempts: %w", attempts, err)
	}

	return nil
}

// SetRemoteUpdateFunc sets the function that will be called when the token is
// rotated, this function is used to update a remote server with the new token.
func (rw *ReadWriter) SetRemoteUpdateFunc(f remoteUpdaterFunc) {
	rw.remoteUpdate = f
}

// Write writes the given token to disk, making sure it has the correct
// permissions, and the correct modification times are set.
func (rw *ReadWriter) Write(id string) error {
	if rw.remoteUpdate != nil {
		if err := rw.remoteUpdate(id); err != nil {
			return err
		}
	}

	err := os.WriteFile(rw.Path, []byte(id), constant.DefaultWorldReadableFileMode)
	if err != nil {
		return fmt.Errorf("write identifier file %q: %w", rw.Path, err)
	}

	// ensure the file is readable by other processes, os.WriteFile does not
	// modify permissions if the file already exists
	if err := rw.setChmod(); err != nil {
		return fmt.Errorf("write identifier file, chmod %q: %w", rw.Path, err)
	}

	// ensure the `mtime` is updated, we have seen tests fail in some versions of
	// Ubuntu because this value is not update when the file is written
	err = os.Chtimes(rw.Path, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("set mtime of identifier file %q: %w", rw.Path, err)
	}

	// make sure we can read the token, and cache the value
	if _, err = rw.Read(); err != nil {
		return fmt.Errorf("read identifier file %q: %w", rw.Path, err)
	}

	return nil
}

func (rw *ReadWriter) setChmod() error {
	return os.Chmod(rw.Path, constant.DefaultWorldReadableFileMode)
}

func (rw *ReadWriter) StartRotation() func() {
	rw.rotationCheckerStarted++

	stopCh := make(chan struct{})

	if rw.rotationCheckerStarted == 1 {
		log.Info().Msg("token rotation is enabled")

		// Create a channel we can use to stop the rotation goroutine.
		rw.rotationStopCh = make(chan struct{})

		go func() {
			// This timer is used to check if the token should be rotated if at
			// least one hour has passed since the last modification of the token
			// file.
			//
			// This is better than using a ticker that ticks every hour because the
			// we can't ensure the tick actually runs every hour (eg: the computer is
			// asleep).
			localCheckDuration := rw.localCheckDuration
			localCheckTicker := time.NewTicker(localCheckDuration)
			defer localCheckTicker.Stop()

			// This timer is used to periodically check if the token is valid. The
			// server might deem a toked as invalid for reasons out of our control,
			// for example if the database is restored to a back-up or if somebody
			// manually invalidates the token in the db.
			remoteCheckDuration := rw.remoteCheckDuration
			remoteCheckTicker := time.NewTicker(remoteCheckDuration)
			defer remoteCheckTicker.Stop()

			for {
				select {
				case <-rw.rotationStopCh:
					log.Info().Msg("token rotation stopped")
					return
				case <-localCheckTicker.C:
					localCheckTicker.Reset(localCheckDuration)

					log.Debug().Msgf("initiating local token check, cached mtime: %s", rw.GetMtime())
					hasChanged, err := rw.HasChanged()
					if err != nil {
						log.Error().Err(err).Msg("error checking if token has changed")
					}

					exp, remain := rw.HasExpired()

					// rotate if the token file has been modified, if the token is
					// expired or if it is very close to expire.
					if hasChanged || exp || remain <= time.Second {
						log.Info().Msg("token TTL expired, rotating token")

						if err := rw.Rotate(); err != nil {
							log.Error().Err(err).Msg("error rotating token")
						}
					} else if remain > 0 && remain < localCheckDuration {
						// check again when the token will expire, which will happen
						// before the next rotation check
						localCheckTicker.Reset(remain)
						log.Debug().Msgf("token will expire soon, checking again in: %s", remain)
					}

				case <-remoteCheckTicker.C:
					log.Debug().Msgf("initiating remote token check after %s", remoteCheckDuration)
					if err := rw.checkTokenFunc(rw.GetCached()); err != nil {
						log.Info().Err(err).Msg("periodic check of token failed, initiating rotation")

						if err := rw.Rotate(); err != nil {
							log.Error().Err(err).Msg("error rotating token")
						}
					}
				}
			}
		}()
	}

	// Start goroutine to handle this caller's stop signal.
	go func() {
		<-stopCh
		rw.rotationCheckerStarted--
		// If all callers have signaled to stop, signal the main rotation goroutine to stop.
		if rw.rotationCheckerStarted == 0 {
			close(rw.rotationStopCh)
		}
	}()

	return func() {
		close(stopCh)
	}
}
