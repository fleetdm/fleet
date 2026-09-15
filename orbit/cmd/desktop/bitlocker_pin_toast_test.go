package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/toast"
	"github.com/stretchr/testify/require"
)

func TestBitLockerPINToast(t *testing.T) {
	t.Parallel()

	const (
		deviceURL        = "https://fleet.example.com/device/token"
		rotatedDeviceURL = "https://fleet.example.com/device/rotated"
	)
	type summary struct {
		needsPIN  bool
		deviceURL string
	}
	type post struct {
		url   string
		popup bool
	}
	showErr := errors.New("PowerShell is blocked")

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
			wantPosts: []post{{deviceURL + "?create_pin=1", true}},
		},
		{
			name:      "a rotated token replaces the toast without popping up again",
			summaries: []summary{{true, deviceURL}, {true, rotatedDeviceURL}},
			wantPosts: []post{{deviceURL + "?create_pin=1", true}, {rotatedDeviceURL + "?create_pin=1", false}},
		},
		{
			name:        "the toast is removed once the PIN is set",
			summaries:   []summary{{true, deviceURL}, {false, deviceURL}, {false, deviceURL}},
			wantPosts:   []post{{deviceURL + "?create_pin=1", true}},
			wantRemoves: 1,
		},
		{
			name:      "nothing is posted or removed when no PIN is needed",
			summaries: []summary{{false, deviceURL}, {false, deviceURL}},
		},
		{
			name:        "a PIN needed again later is posted without popping up",
			summaries:   []summary{{true, deviceURL}, {false, deviceURL}, {true, deviceURL}},
			wantPosts:   []post{{deviceURL + "?create_pin=1", true}, {deviceURL + "?create_pin=1", false}},
			wantRemoves: 1,
		},
		{
			name:      "a failed post is retried only when the link changes",
			summaries: []summary{{true, deviceURL}, {true, deviceURL}, {true, rotatedDeviceURL}, {false, rotatedDeviceURL}},
			failShow:  true,
			wantPosts: []post{{deviceURL + "?create_pin=1", true}, {rotatedDeviceURL + "?create_pin=1", true}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var posts []post
			var removes int
			pinToast := &bitLockerPINToast{
				show: func(n toast.Notification) error {
					require.Equal(t, bitLockerPINToastTag, n.Tag)
					require.Equal(t, bitLockerPINToastGroup, n.Group)
					require.Equal(t, bitLockerPINToastTitle, n.Title)
					require.Equal(t, bitLockerPINToastBody, n.Body)
					require.Equal(t, bitLockerPINToastButton, n.ButtonLabel)
					require.Equal(t, bitLockerPINToastLifetime, n.ExpiresIn)
					posts = append(posts, post{url: n.URL, popup: !n.SuppressPopup})
					if tc.failShow {
						return showErr
					}
					return nil
				},
				remove: func(tag, group string) error {
					require.Equal(t, bitLockerPINToastTag, tag)
					require.Equal(t, bitLockerPINToastGroup, group)
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

	const deviceURL = "https://fleet.example.com/device/token"

	for _, tc := range []struct {
		name string
		// markerAge is how long ago an earlier process popped up the toast, or zero for no marker.
		markerAge       time.Duration
		needsPIN        bool
		wantPopups      []bool
		wantRemoves     int
		wantMarker      bool
		wantMarkerMoved bool
	}{
		{name: "a first popup writes the marker", needsPIN: true, wantPopups: []bool{true}, wantMarker: true},
		{name: "a restart soon after a popup posts silently", markerAge: 10 * time.Minute, needsPIN: true, wantPopups: []bool{false}, wantMarker: true},
		{
			name: "a restart after the last toast expired pops up again", markerAge: 2 * time.Hour, needsPIN: true,
			wantPopups: []bool{true}, wantMarker: true, wantMarkerMoved: true,
		},
		{name: "a toast from an earlier process is removed once the PIN is set", markerAge: 10 * time.Minute, wantRemoves: 1},
		{name: "a host that never showed the toast starts nothing", needsPIN: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			markerPath := filepath.Join(t.TempDir(), "bitlocker-pin-toast")
			markerTime := time.Now().Add(-tc.markerAge).Truncate(time.Second)
			if tc.markerAge > 0 {
				require.NoError(t, os.WriteFile(markerPath, nil, 0o600))
				require.NoError(t, os.Chtimes(markerPath, markerTime, markerTime))
			}

			var popups []bool
			var removes int
			pinToast := newBitLockerPINToast(markerPath)
			pinToast.show = func(n toast.Notification) error {
				popups = append(popups, !n.SuppressPopup)
				return nil
			}
			pinToast.remove = func(string, string) error {
				removes++
				return nil
			}

			pinToast.reconcile(tc.needsPIN, deviceURL)

			require.Equal(t, tc.wantPopups, popups)
			require.Equal(t, tc.wantRemoves, removes)
			info, err := os.Stat(markerPath)
			if !tc.wantMarker {
				require.ErrorIs(t, err, fs.ErrNotExist)
				return
			}
			require.NoError(t, err)
			if tc.markerAge > 0 {
				require.Equal(t, tc.wantMarkerMoved, !info.ModTime().Equal(markerTime), "a popup moves the marker's time and a silent post does not")
			}
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
					pinToast.submit(needsPIN, "https://fleet.example.com/device/token")
				}
				pinToast.mu.Unlock()
				synctest.Wait()

				require.Equal(t, tc.wantPosts, posts)
				require.Equal(t, tc.wantRemoves, removes)
			})
		})
	}
}
