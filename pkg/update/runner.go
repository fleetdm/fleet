package update

import (
	"bytes"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// RunnerOptions is options provided for the update runner.
type RunnerOptions struct {
	// CheckInterval is the interval to check for updates.
	CheckInterval time.Duration
	// Targets is the names of the artifacts to watch for updates.
	Targets map[string]string
}

// Runner is a specialized runner for the updater. It is designed with Execute and
// Interrupt functions to be compatible with oklog/run.
type Runner struct {
	client    *Updater
	opt       RunnerOptions
	cancel    chan struct{}
	hashCache map[string][]byte
}

// NewRunner creates a new runner with the provided options. The runner must be
// started with Execute.
func NewRunner(client *Updater, opt RunnerOptions) (*Runner, error) {
	if opt.CheckInterval <= 0 {
		return nil, errors.New("Runner must be configured with interval greater than 0")
	}
	if len(opt.Targets) == 0 {
		return nil, errors.New("Runner must have nonempty subscriptions")
	}

	// Initialize hash cache
	cache := make(map[string][]byte)
	for target, channel := range opt.Targets {
		meta, err := client.Lookup(target, channel)
		if err != nil {
			return nil, errors.Wrap(err, "initialize update cache")
		}

		_, hash, err := selectHashFunction(meta)
		if err != nil {
			return nil, errors.Wrap(err, "select hash for cache")
		}
		cache[target] = hash
	}

	return &Runner{
		client: client,
		opt:    opt,

		// chan gets capacity of 1 so we don't end up hung if Interrupt is
		// called after Execute has already returned.
		cancel:    make(chan struct{}, 1),
		hashCache: cache,
	}, nil
}

// Execute begins a loop checking for updates.
func (r *Runner) Execute() error {
	ticker := time.NewTicker(r.opt.CheckInterval)
	defer ticker.Stop()

	// Run until cancel or returning an error
	for {
		select {
		case <-r.cancel:
			return nil

		case <-ticker.C:
			// On each tick, check for updates
			didUpdate, err := r.updateAction()
			if err != nil {
				log.Info().Err(err).Msg("update failed")
			}
			if didUpdate {
				log.Info().Msg("exiting due to successful update")
				return nil
			}
		}
	}
}

func (r *Runner) updateAction() (bool, error) {
	var didUpdate bool
	if err := r.client.UpdateMetadata(); err != nil {
		// Consider this a non-fatal error since it will be common to be offline
		// or otherwise unable to retrieve the metadata.
		return didUpdate, errors.Wrap(err, "update metadata")
	}

	for target, channel := range r.opt.Targets {
		meta, err := r.client.Lookup(target, channel)
		if err != nil {
			return didUpdate, errors.Wrapf(err, "lookup failed")
		}

		// Check whether the hash has changed
		_, hash, err := selectHashFunction(meta)
		if err != nil {
			return didUpdate, errors.Wrap(err, "select hash for cache")
		}

		if !bytes.Equal(r.hashCache[target], hash) {
			// Update detected
			log.Info().Str("target", target).Str("channel", channel).Msg("update detected")
			if err := r.updateTarget(target, channel); err != nil {
				return didUpdate, errors.Wrapf(err, "update %s@%s", target, channel)
			}
			log.Info().Str("target", target).Str("channel", channel).Msg("update completed")
			didUpdate = true
		} else {
			log.Debug().Str("target", target).Str("channel", channel).Msg("no update")
		}
	}

	return didUpdate, nil
}

func (r *Runner) updateTarget(target, channel string) error {
	path, err := r.client.Get(target, channel)
	if err != nil {
		return errors.Wrap(err, "get binary")
	}

	if target != "orbit" {
		return nil
	}

	// Symlink Orbit binary
	linkPath := filepath.Join(r.client.opt.RootDirectory, "bin", "orbit", filepath.Base(path))
	// Rename the old file otherwise overwrite fails
	if err := os.Rename(linkPath, linkPath+".old"); err != nil {
		return errors.Wrap(err, "move old symlink current")
	}
	if err := os.Symlink(path, linkPath); err != nil {
		return errors.Wrap(err, "symlink current")
	}

	return nil
}

func (r *Runner) Interrupt(err error) {
	r.cancel <- struct{}{}
	log.Debug().Msg("interrupt updater")
}
