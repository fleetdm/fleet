package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/fleetdm/fleet/v4/server/service"
)

type client interface {
	CheckToken(string) error
}

type reader interface {
	Read() (string, error)
	GetCached() string
}

// TODO: docs
type RemoteChecker struct {
	reader reader
	client client
}

// TODO: docs
func NewChecker(path string, client client) *RemoteChecker {
	return &RemoteChecker{
		reader: &Reader{Path: path},
		client: client,
	}
}

func (c *RemoteChecker) isValid(err error) bool {
	return err == nil || errors.Is(err, service.ErrMissingLicense)
}

// TODO: docs
func (c *RemoteChecker) AwaitValid() {
	// TODO: find appropriate values
	// randomized interval = RetryInterval * (random value in range [1 - RandomizationFactor, 1 + RandomizationFactor])
	retryStrategy := backoff.NewExponentialBackOff()
	retryStrategy.InitialInterval = 2 * time.Second
	retryStrategy.MaxElapsedTime = 1 * time.Minute

	for {
		if err := backoff.Retry(
			func() error {
				if _, err := c.reader.Read(); err != nil {
					return err
				}
				err := c.client.CheckToken(c.reader.GetCached())
				if !c.isValid(err) {
					// TODO: better error
					return errors.New("invalid")
				}
				return nil
			},
			retryStrategy,
		); err != nil {
			// TODO: what do we do here?
			fmt.Println("backoff gave up")
			continue
		}

		return
	}
}
