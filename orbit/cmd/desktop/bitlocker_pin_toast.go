package main

import (
	"errors"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/toast"
	"github.com/rs/zerolog/log"
)

const (
	bitLockerPINToastTag   = "bitlocker-pin"
	bitLockerPINToastGroup = "fleet-desktop"
	// createPINQuery is what tells the My device page to open the Create PIN modal.
	createPINQuery = "?create_pin=1"
	// bitLockerPINToastLifetime clears an ignored toast from Notification Center before its link stops working.
	bitLockerPINToastLifetime = time.Hour

	bitLockerPINToastTitle  = "Set your BitLocker PIN to protect this device"
	bitLockerPINToastBody   = "Your IT team requires a BitLocker PIN on this device. Set yours now to keep your files safe if your device is ever lost."
	bitLockerPINToastButton = "Create PIN"
)

// bitLockerPINToast asks the end user, through a Windows toast, to create the BitLocker startup PIN their fleet requires.
// It pops up once per Windows login, and later posts only keep the Notification Center copy's link current. Orbit also
// restarts Fleet Desktop within a login, so a per-user marker file records the login that last saw the popup. Failures are
// logged and swallowed, because the My device banner remains.
type bitLockerPINToast struct {
	show   func(toast.Notification) error
	remove func(tag, group string) error
	// markerPath exists while a toast may be in Notification Center, and holds the login that last saw the popup. Empty
	// disables it.
	markerPath string
	// loginID identifies the current Windows login, empty if it could not be read.
	loginID string

	// submitted numbers each summary handed to submit.
	submitted atomic.Uint64

	mu       sync.Mutex
	poppedUp bool
	// attemptedURL is the link of the last post attempt.
	attemptedURL string
	// posted is whether a toast may be in Notification Center, so hosts that never showed one never start PowerShell.
	posted bool
}

func newBitLockerPINToast(markerPath, loginID string) *bitLockerPINToast {
	t := &bitLockerPINToast{show: toast.Show, remove: toast.Remove, markerPath: markerPath, loginID: loginID}
	if markerPath == "" {
		return t
	}
	if marker, err := os.ReadFile(markerPath); err == nil {
		t.posted = true
		// An unreadable login pops up the toast again
		t.poppedUp = loginID != "" && string(marker) == loginID
	}
	return t
}

// submit reconciles the toast with a Fleet Desktop summary in the background, because PowerShell can take seconds to start.
// Waiting for the lock does not preserve order, so an update that gets it after a newer summary was submitted is dropped.
func (t *bitLockerPINToast) submit(needsPIN bool, deviceURL string) {
	seq := t.submitted.Add(1)
	go func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		if seq != t.submitted.Load() {
			return
		}
		t.reconcile(needsPIN, deviceURL)
	}()
}

// reconcile brings the toast in line with a summary. The caller holds lock. deviceURL carries the device token.
func (t *bitLockerPINToast) reconcile(needsPIN bool, deviceURL string) {
	if !needsPIN {
		t.attemptedURL = ""
		if !t.posted {
			return
		}
		t.posted = false
		if err := t.remove(bitLockerPINToastTag, bitLockerPINToastGroup); err != nil {
			log.Warn().Err(err).Msg("could not remove the BitLocker PIN toast")
		}
		// The toast expires on its own, so a failed removal is not retried.
		if t.markerPath != "" {
			if err := os.Remove(t.markerPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				log.Warn().Err(err).Msg("could not delete the BitLocker PIN toast marker")
			}
		}
		return
	}

	link := deviceURL + createPINQuery
	if link == t.attemptedURL {
		return
	}
	t.attemptedURL = link
	popup := !t.poppedUp
	err := t.show(toast.Notification{
		Tag:           bitLockerPINToastTag,
		Group:         bitLockerPINToastGroup,
		Title:         bitLockerPINToastTitle,
		Body:          bitLockerPINToastBody,
		ButtonLabel:   bitLockerPINToastButton,
		URL:           link,
		ExpiresIn:     bitLockerPINToastLifetime,
		SuppressPopup: !popup,
		StayOnScreen:  true,
	})
	if err != nil {
		// An orbit too old to register the notification identity may happen during an upgrade, and it registers on its next start.
		if errors.Is(err, toast.ErrAppIDNotRegistered) {
			log.Warn().Msg("skipped the BitLocker PIN toast, orbit has not registered the notification identity")
		} else {
			log.Error().Err(err).Msg("could not show the BitLocker PIN toast")
		}
		return
	}
	log.Info().Bool("popup", popup).Msg("posted the BitLocker PIN toast")
	t.poppedUp = true
	t.posted = true
	if err := t.writeMarker(); err != nil {
		log.Warn().Err(err).Msg("could not write the BitLocker PIN toast marker")
	}
}

// writeMarker records that this login has seen the popup and that a toast may be in Notification Center.
func (t *bitLockerPINToast) writeMarker() error {
	if t.markerPath == "" {
		return nil
	}
	return os.WriteFile(t.markerPath, []byte(t.loginID), 0o600)
}
