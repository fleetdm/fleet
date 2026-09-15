package main

import (
	"errors"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/toast"
	"github.com/rs/zerolog/log"
)

const (
	bitLockerPINToastTag   = "bitlocker-pin"
	bitLockerPINToastGroup = "fleet-desktop"
	// bitLockerPINToastLifetime clears an ignored toast from Notification Center before its link stops working. The server
	// accepts a device token until an hour after it is rotated, and the toast is re-posted with the new token.
	bitLockerPINToastLifetime = time.Hour

	bitLockerPINToastTitle  = "Set your BitLocker PIN to protect this device"
	bitLockerPINToastBody   = "Your IT team requires a BitLocker PIN on this device. Set yours now to keep your files safe if your device is ever lost."
	bitLockerPINToastButton = "Create PIN"
)

// bitLockerPINToast asks the end user, through a Windows toast, to create the BitLocker startup PIN their fleet requires.
// It pops up at most once per Fleet Desktop process, and later posts only keep the Notification Center copy's link
// current. Orbit restarts Fleet Desktop whenever its run group restarts, not only at login, so a per-user marker file
// carries the last popup across processes. Failures are logged and swallowed, because the My device banner remains.
type bitLockerPINToast struct {
	show   func(toast.Notification) error
	remove func(tag, group string) error
	// markerPath exists while a toast may be in Notification Center, and its modification time is the last popup. Empty
	// disables it.
	markerPath string

	mu       sync.Mutex
	poppedUp bool
	// attemptedURL is the link of the last post attempt. A failed post is retried only when the link changes, so a host
	// that blocks PowerShell does not start it on every summary poll.
	attemptedURL string
	// posted is whether a toast may be in Notification Center, so hosts that never showed one never start PowerShell.
	posted bool
}

func newBitLockerPINToast(markerPath string) *bitLockerPINToast {
	t := &bitLockerPINToast{show: toast.Show, remove: toast.Remove, markerPath: markerPath}
	if markerPath == "" {
		return t
	}
	if info, err := os.Stat(markerPath); err == nil {
		t.posted = true
		// A restart within the lifetime of the last popup's toast would otherwise pop it up again.
		t.poppedUp = time.Since(info.ModTime()) < bitLockerPINToastLifetime
	}
	return t
}

// update reconciles the toast with a Fleet Desktop summary. deviceURL carries the device token, so it is never logged.
func (t *bitLockerPINToast) update(needsPIN bool, deviceURL string) {
	t.mu.Lock()
	defer t.mu.Unlock()

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

	link := deviceURL + "?create_pin=1"
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
	})
	if err != nil {
		log.Warn().Err(err).Msg("could not show the BitLocker PIN toast")
		return
	}
	log.Info().Bool("popup", popup).Msg("posted the BitLocker PIN toast")
	t.poppedUp = true
	t.posted = true
	if err := t.touchMarker(popup); err != nil {
		log.Warn().Err(err).Msg("could not write the BitLocker PIN toast marker")
	}
}

// touchMarker creates the marker if needed, and moves its modification time only for a popup.
func (t *bitLockerPINToast) touchMarker(popup bool) error {
	if t.markerPath == "" {
		return nil
	}
	f, err := os.OpenFile(t.markerPath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if !popup {
		return nil
	}
	now := time.Now()
	return os.Chtimes(t.markerPath, now, now)
}
