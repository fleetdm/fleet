package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/fleetdm/fleet/v4/orbit/pkg/toast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	deviceURL        = "https://fleet.example.com/device/token"
	rotatedDeviceURL = "https://fleet.example.com/device/rotated"
)

func TestBitLockerPINToast(t *testing.T) {
	t.Parallel()

	type summary struct {
		needsPIN  bool
		deviceURL string
	}
	type post struct {
		url   string
		popup bool
	}
	showErr := errors.New("PowerShell is blocked")
	link, rotatedLink := deviceURL+createPINQuery, rotatedDeviceURL+createPINQuery

	for _, tc := range []struct {
		name        string
		summaries   []summary
		failShow    bool
		wantPosts   []post
		wantRemoves int
	}{
		{
			name:      "pops up once while the PIN stays needed",
			summaries: []summary{{true, deviceURL}, {true, deviceURL}, {true, deviceURL}},
			wantPosts: []post{{link, true}},
		},
		{
			name:      "a rotated token replaces the toast without popping up again",
			summaries: []summary{{true, deviceURL}, {true, rotatedDeviceURL}},
			wantPosts: []post{{link, true}, {rotatedLink, false}},
		},
		{
			name:        "the toast is removed once the PIN is set",
			summaries:   []summary{{true, deviceURL}, {false, deviceURL}, {false, deviceURL}},
			wantPosts:   []post{{link, true}},
			wantRemoves: 1,
		},
		{
			name:      "nothing is posted or removed when no PIN is needed",
			summaries: []summary{{false, deviceURL}, {false, deviceURL}},
		},
		{
			name:        "a PIN needed again later is posted without popping up",
			summaries:   []summary{{true, deviceURL}, {false, deviceURL}, {true, deviceURL}},
			wantPosts:   []post{{link, true}, {link, false}},
			wantRemoves: 1,
		},
		{
			name:      "a failed post is retried only when the link changes",
			summaries: []summary{{true, deviceURL}, {true, deviceURL}, {true, rotatedDeviceURL}, {false, rotatedDeviceURL}},
			failShow:  true,
			wantPosts: []post{{link, true}, {rotatedLink, true}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var posts []post
			var removes int
			pinToast := &bitLockerPINToast{
				show: func(n toast.Notification) error {
					assert.Equal(t, bitLockerPINToastTitle, n.Title)
					assert.Equal(t, bitLockerPINToastBody, n.Body)
					posts = append(posts, post{url: n.URL, popup: !n.SuppressPopup})
					if tc.failShow {
						return showErr
					}
					return nil
				},
				remove: func(tag, group string) error {
					assert.Equal(t, bitLockerPINToastTag, tag)
					assert.Equal(t, bitLockerPINToastGroup, group)
					removes++
					return nil
				},
			}

			for _, s := range tc.summaries {
				pinToast.reconcile(s.needsPIN, s.deviceURL)
			}

			require.Equal(t, tc.wantPosts, posts)
			require.Equal(t, tc.wantRemoves, removes)
		})
	}
}

func TestBitLockerPINToastAcrossProcesses(t *testing.T) {
	t.Parallel()

	const (
		loginID      = "1:134036577000000000"
		earlierLogin = "1:134036001000000000"
	)

	for _, tc := range []struct {
		name string
		// marker is what an earlier Fleet Desktop process left, or nil for none.
		marker *string
		// noMarkerPath disables the marker, as when Fleet Desktop cannot find the directory for it.
		noMarkerPath bool
		loginID      string
		needsPIN     bool
		failShow     bool
		// wantPopups lists whether each post popped up.
		wantPopups  []bool
		wantRemoves int
		// wantMarker is the marker's content afterwards, or nil if it should not exist.
		wantMarker *string
	}{
		{name: "a first popup records the login", loginID: loginID, needsPIN: true, wantPopups: []bool{true}, wantMarker: new(loginID)},
		{
			name: "Fleet Desktop restarting within a login posts silently", marker: new(loginID), loginID: loginID, needsPIN: true,
			wantPopups: []bool{false}, wantMarker: new(loginID),
		},
		{
			name: "a new login pops up again", marker: new(earlierLogin), loginID: loginID, needsPIN: true,
			wantPopups: []bool{true}, wantMarker: new(loginID),
		},
		{name: "an unreadable login pops up", marker: new(""), needsPIN: true, wantPopups: []bool{true}, wantMarker: new("")},
		{name: "a toast from an earlier process is removed once the PIN is set", marker: new(loginID), loginID: loginID, wantRemoves: 1},
		{name: "a host that never showed the toast starts nothing", loginID: loginID},
		// Without a marker every process pops up again, which is better than a prompt nobody sees.
		{name: "no marker to write pops up anyway", noMarkerPath: true, loginID: loginID, needsPIN: true, wantPopups: []bool{true}},
		// A toast that was never shown must not leave a marker claiming this login saw one.
		{name: "a failed post records nothing", loginID: loginID, needsPIN: true, failShow: true, wantPopups: []bool{true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			markerPath := filepath.Join(t.TempDir(), "bitlocker-pin-toast")
			if tc.marker != nil {
				require.NoError(t, os.WriteFile(markerPath, []byte(*tc.marker), 0o600))
			}
			constructedPath := markerPath
			if tc.noMarkerPath {
				constructedPath = ""
			}

			var popups []bool
			var removes int
			pinToast := newBitLockerPINToast(constructedPath, tc.loginID)
			pinToast.show = func(n toast.Notification) error {
				popups = append(popups, !n.SuppressPopup)
				if tc.failShow {
					return errors.New("PowerShell is blocked")
				}
				return nil
			}
			pinToast.remove = func(string, string) error {
				removes++
				return nil
			}

			pinToast.reconcile(tc.needsPIN, deviceURL)

			require.Equal(t, tc.wantPopups, popups)
			require.Equal(t, tc.wantRemoves, removes)
			marker, err := os.ReadFile(markerPath)
			if tc.wantMarker == nil {
				require.ErrorIs(t, err, fs.ErrNotExist)
				return
			}
			require.NoError(t, err)
			require.Equal(t, *tc.wantMarker, string(marker))
		})
	}
}

func TestBitLockerPINToastSubmitDropsStaleUpdates(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		summaries   []bool
		wantPosts   int
		wantRemoves int
	}{
		// Without the sequence check, the older needsPIN=true update could run last and post a toast for a PIN already set.
		{name: "a PIN set while an update waited leaves no toast", summaries: []bool{true, false}},
		{name: "a PIN needed again while an update waited posts once", summaries: []bool{false, true}, wantPosts: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var posts, removes int
				pinToast := &bitLockerPINToast{
					show:   func(toast.Notification) error { posts++; return nil },
					remove: func(string, string) error { removes++; return nil },
				}

				// Hold the lock so every submitted update is waiting on it at once, as when PowerShell is slow.
				pinToast.mu.Lock()
				for _, needsPIN := range tc.summaries {
					pinToast.submit(needsPIN, deviceURL)
				}
				pinToast.mu.Unlock()
				synctest.Wait()

				require.Equal(t, tc.wantPosts, posts)
				require.Equal(t, tc.wantRemoves, removes)
			})
		})
	}
}
