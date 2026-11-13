package mdmtest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/micromdm/plist"
	"github.com/stretchr/testify/require"
)

func TestApplePushTopic(t *testing.T) {
	var (
		mu     sync.Mutex
		topics []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var msg struct {
			Topic string
		}
		if err := plist.Unmarshal(body, &msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		topics = append(topics, msg.Topic)
		mu.Unlock()
	}))
	t.Cleanup(srv.Close)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test device"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}, &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test device"}}, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	newDevice := func(opts ...TestMDMAppleClientOption) *TestAppleMDMClient {
		c := NewTestMDMClientAppleDirect(AppleEnrollInfo{MDMURL: srv.URL}, "MacBookPro16,1", opts...)
		c.scepCert, c.scepKey = cert, key
		return c
	}
	// sendCheckins sends a sample of check-in and command-report messages and
	// returns the topics the server received.
	sendCheckins := func(t *testing.T, c *TestAppleMDMClient) []string {
		mu.Lock()
		topics = nil
		mu.Unlock()

		require.NoError(t, c.Authenticate())
		require.NoError(t, c.TokenUpdate(false))
		_, err := c.Idle()
		require.NoError(t, err)
		_, err = c.Acknowledge("cmd-uuid")
		require.NoError(t, err)
		require.NoError(t, c.Checkout())

		mu.Lock()
		defer mu.Unlock()
		require.Len(t, topics, 5)
		return topics
	}

	t.Run("default topic is unique per device", func(t *testing.T) {
		a, b := newDevice(), newDevice()
		require.NotEqual(t, a.PushTopic(), b.PushTopic())
		for _, got := range sendCheckins(t, a) {
			require.Equal(t, "com.apple.mgmt.External."+a.UUID, got)
		}
	})

	t.Run("WithPushTopic is sent by every device", func(t *testing.T) {
		const topic = "com.apple.mgmt.External.10ac3ce5-4668-4e58-b69a-b2b5ce667589"
		for range 2 {
			c := newDevice(WithPushTopic(topic))
			require.Equal(t, topic, c.PushTopic())
			for _, got := range sendCheckins(t, c) {
				require.Equal(t, topic, got)
			}
		}
	})
}
