package main

import (
	"net/url"
	"path"
	"sync"
	"time"
)

type tokenUpdater struct {
	// mutable fields, protected by mutex:
	mu sync.Mutex

	// current known valid token
	currentToken string
	// path to the token file
	tokenFile string
	// last mtime of the token file from when the currentToken was loaded
	lastMtime time.Time
	// list of registered notification channels when rotation is detected:
	// immediately when a rotation is detected, all channels receive a channel
	// that is closed when the token has been properly reloaded and validated.
	notifyChans []chan<- chan struct{}

	// immutable fields:

	// old orbit versions did not support rotation, set to false when such an orbit is detected
	// Note that when rotation is not supported, the currentToken field above is also immutable.
	rotationSupported bool
	// the device URL, without the last fragment with the token
	partialDeviceURL url.URL
}

func NewRotatingTokenUpdater(tokenFile, partialURL string) (*tokenUpdater, error) {
	parsed, err := url.Parse(partialURL)
	if err != nil {
		return nil, err
	}

	tu := &tokenUpdater{
		tokenFile:         tokenFile,
		partialDeviceURL:  *parsed,
		rotationSupported: true,
	}
	// TODO: start goroutine to check changes to the file
	return tu, nil
}

func NewStaticTokenUpdater(fullURL string) (*tokenUpdater, error) {
	parsed, err := url.Parse(fullURL)
	if err != nil {
		return nil, err
	}
	tok := path.Base(parsed.Path)
	parsed.Path = path.Dir(parsed.Path)
	return &tokenUpdater{
		rotationSupported: false,
		partialDeviceURL:  *parsed,
		currentToken:      tok,
	}, nil
}

func (tu *tokenUpdater) Notify(c chan<- chan struct{}) {
	if !tu.rotationSupported {
		return
	}

	tu.mu.Lock()
	defer tu.mu.Unlock()
	tu.notifyChans = append(tu.notifyChans, c)
}

func (tu *tokenUpdater) DeviceURL(syncCheck bool) string {
	if !tu.rotationSupported {
		devURL := tu.partialDeviceURL
		devURL.Path = path.Join(devURL.Path, tu.currentToken)
		return devURL.String()
	}
	panic("unimplemented")
}

func (tu *tokenUpdater) TransparencyURL(syncCheck bool) string {
	if !tu.rotationSupported {
		tURL := tu.partialDeviceURL
		tURL.Path = "/api/latest/fleet/device/" + tu.currentToken + "/transparency"
		return tURL.String()
	}
	panic("unimplemented")
}
