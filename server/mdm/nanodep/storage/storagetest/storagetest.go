// Package storagetest offers a battery of tests for storage.AllStorage implementations.
package storagetest

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
)

// Run runs a battery of tests on the storage.AllStorage returned by storageFn.
func Run(t *testing.T, storageFn func(t *testing.T) storage.AllDEPStorage) {
	ctx := context.Background()

	// Test retrieval methods on empty storage.
	t.Run("empty", func(t *testing.T) {
		const name = "empty"

		s := storageFn(t)

		pemCert, pemKey, err := s.RetrieveTokenPKI(ctx, name)
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("unexpected error: %s", err)
		}
		if pemCert != nil {
			t.Fatal("expected nil cert pem")
		}
		if pemKey != nil {
			t.Fatal("expected nil key pem")
		}

		tokens, err := s.RetrieveAuthTokens(ctx, name)
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("unexpected error: %s", err)
		}
		if tokens != nil {
			t.Fatal("expected nil tokens")
		}

		config, err := s.RetrieveConfig(ctx, name)
		checkErr(t, err)
		emptyConfig := client.Config{}
		if config == nil || *config != emptyConfig {
			t.Fatalf("expected empty config: %+v", config)
		}

		// Profile assigner storing and retrieval.
		profileUUID, modTime, err := s.RetrieveAssignerProfile(ctx, name)
		checkErr(t, err)
		if profileUUID != "" {
			t.Fatal("expected empty profileUUID")
		}
		if !modTime.IsZero() {
			t.Fatal("expected zero modTime")
		}

		cursor, cursorAt, err := s.RetrieveCursor(ctx, name)
		checkErr(t, err)
		if cursor != "" {
			t.Fatal("expected empty cursor")
		}
		if !cursorAt.IsZero() {
			t.Fatal("expected empty cursor at")
		}
	})

	testWithName := func(t *testing.T, name string, s storage.AllDEPStorage) {

		// PKI storing and retrieval.
		pemCert, pemKey, err := s.RetrieveTokenPKI(ctx, name)
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("unexpected error: %s", err)
		}
		if err == nil {
			t.Fatal("expected error")
		}
		if pemCert != nil {
			t.Fatal("expected nil cert pem")
		}
		if pemKey != nil {
			t.Fatal("expected nil key pem")
		}
		pemCert, pemKey = generatePKI(t, "basicdn", 1)
		err = s.StoreTokenPKI(ctx, name, pemCert, pemKey)
		checkErr(t, err)
		pemCert2, pemKey2, err := s.RetrieveTokenPKI(ctx, name)
		checkErr(t, err)
		if !bytes.Equal(pemCert, pemCert2) {
			t.Fatalf("pem cert mismatch: %s vs. %s", pemCert, pemCert2)
		}
		if !bytes.Equal(pemKey, pemKey2) {
			t.Fatalf("pem key mismatch: %s vs. %s", pemKey, pemKey2)
		}

		// Token storing and retrieval.
		tokens, err := s.RetrieveAuthTokens(ctx, name)
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("unexpected error: %s", err)
		}
		if tokens != nil {
			t.Fatal("expected nil tokens")
		}
		tokens = &client.OAuth1Tokens{
			ConsumerKey:       "CK_9af2f8218b150c351ad802c6f3d66abe",
			ConsumerSecret:    "CS_9af2f8218b150c351ad802c6f3d66abe",
			AccessToken:       "AT_9af2f8218b150c351ad802c6f3d66abe",
			AccessSecret:      "AS_9af2f8218b150c351ad802c6f3d66abe",
			AccessTokenExpiry: time.Now().UTC(),
		}
		err = s.StoreAuthTokens(ctx, name, tokens)
		checkErr(t, err)
		tokens2, err := s.RetrieveAuthTokens(ctx, name)
		checkErr(t, err)
		checkTokens(t, tokens, tokens2)
		tokens3 := &client.OAuth1Tokens{
			ConsumerKey:       "foo_CK_9af2f8218b150c351ad802c6f3d66abe",
			ConsumerSecret:    "foo_CS_9af2f8218b150c351ad802c6f3d66abe",
			AccessToken:       "foo_AT_9af2f8218b150c351ad802c6f3d66abe",
			AccessSecret:      "foo_AS_9af2f8218b150c351ad802c6f3d66abe",
			AccessTokenExpiry: time.Now().Add(5 * time.Second).UTC(),
		}
		err = s.StoreAuthTokens(ctx, name, tokens3)
		checkErr(t, err)
		tokens4, err := s.RetrieveAuthTokens(ctx, name)
		checkErr(t, err)
		checkTokens(t, tokens3, tokens4)

		// Config storing and retrieval.
		config, err := s.RetrieveConfig(ctx, name)
		checkErr(t, err)
		emptyConfig := client.Config{}
		if config == nil || *config != emptyConfig {
			t.Fatalf("expected empty config: %+v", config)
		}
		config = &client.Config{
			BaseURL: "https://config.example.com",
		}
		err = s.StoreConfig(ctx, name, config)
		checkErr(t, err)
		config2, err := s.RetrieveConfig(ctx, name)
		checkErr(t, err)
		if *config != *config2 {
			t.Fatalf("config mismatch: %+v vs. %+v", config, config2)
		}
		config2 = &client.Config{
			BaseURL: "https://config2.example.com",
		}
		err = s.StoreConfig(ctx, name, config2)
		checkErr(t, err)
		config3, err := s.RetrieveConfig(ctx, name)
		checkErr(t, err)
		if *config2 != *config3 {
			t.Fatalf("config mismatch: %+v vs. %+v", config2, config3)
		}

		// Profile assigner storing and retrieval.
		profileUUID, modTime, err := s.RetrieveAssignerProfile(ctx, name)
		checkErr(t, err)
		if profileUUID != "" {
			t.Fatal("expected empty profileUUID")
		}
		if !modTime.IsZero() {
			t.Fatal("expected zero modTime")
		}
		profileUUID = "43277A13FBCA0CFC"
		err = s.StoreAssignerProfile(ctx, name, profileUUID)
		checkErr(t, err)
		profileUUID2, modTime, err := s.RetrieveAssignerProfile(ctx, name)
		checkErr(t, err)
		if profileUUID != profileUUID2 {
			t.Fatalf("profileUUID mismatch: %s vs. %s", profileUUID, profileUUID2)
		}
		now := time.Now()
		if modTime.Before(now.Add(-1*time.Minute)) || modTime.After(now.Add(1*time.Minute)) {
			t.Fatalf("mismatch modTime, expected: %s (+/- 1m), actual: %s", now, modTime)
		}
		time.Sleep(1 * time.Second)
		profileUUID3 := "foo_43277A13FBCA0CFC"
		err = s.StoreAssignerProfile(ctx, name, profileUUID3)
		checkErr(t, err)
		profileUUID4, modTime2, err := s.RetrieveAssignerProfile(ctx, name)
		checkErr(t, err)
		if profileUUID3 != profileUUID4 {
			t.Fatalf("profileUUID mismatch: %s vs. %s", profileUUID, profileUUID3)
		}
		if modTime2 == modTime {
			t.Fatalf("expected time update: %s", modTime2)
		}
		now = time.Now()
		if modTime2.Before(now.Add(-1*time.Minute)) || modTime2.After(now.Add(1*time.Minute)) {
			t.Fatalf("mismatch modTime, expected: %s (+/- 1m), actual: %s", now, modTime)
		}

		cursor, modTime, err := s.RetrieveCursor(ctx, name)
		checkErr(t, err)
		if cursor != "" {
			t.Fatal("expected empty cursor")
		}
		if !modTime.IsZero() {
			t.Fatal("expected empty cursor at")
		}
		cursor = "MTY1NzI2ODE5Ny0x"
		err = s.StoreCursor(ctx, name, cursor)
		checkErr(t, err)
		cursor2, modTime2, err := s.RetrieveCursor(ctx, name)
		checkErr(t, err)
		if cursor != cursor2 {
			t.Fatalf("cursor mismatch: %s vs. %s", cursor, cursor2)
		}
		if modTime2.IsZero() {
			t.Fatalf("expected cursor at to not be zero")
		}
		if now := time.Now(); modTime2.Before(now.Add(-1*time.Minute)) || modTime2.After(now.Add(1*time.Minute)) {
			t.Fatalf("expected cursor at to be within bounds")
		}
		cursor2 = "foo_MTY1NzI2ODE5Ny0x"
		err = s.StoreCursor(ctx, name, cursor2)
		checkErr(t, err)
		cursor3, modTime3, err := s.RetrieveCursor(ctx, name)
		checkErr(t, err)
		if cursor2 != cursor3 {
			t.Fatalf("cursor mismatch: %s vs. %s", cursor2, cursor3)
		}
		if modTime3.Before(modTime2) {
			t.Fatalf("cursor at should be later than previous")
		}
	}

	t.Run("basic", func(t *testing.T) {
		storage := storageFn(t)
		testWithName(t, "basic", storage)
	})

	t.Run("multiple-names", func(t *testing.T) {
		storage := storageFn(t)
		testWithName(t, "name1", storage)
		testWithName(t, "name2", storage)
	})
}

func checkTokens(t *testing.T, t1 *client.OAuth1Tokens, t2 *client.OAuth1Tokens) {
	if t1.ConsumerKey != t2.ConsumerKey {
		t.Fatalf("tokens consumer_key mismatch: %s vs. %s", t1.ConsumerKey, t2.ConsumerKey)
	}
	if t1.ConsumerSecret != t2.ConsumerSecret {
		t.Fatalf("tokens consumer_secret mismatch: %s vs. %s", t1.ConsumerSecret, t2.ConsumerSecret)
	}
	if t1.AccessToken != t2.AccessToken {
		t.Fatalf("tokens access_token mismatch: %s vs. %s", t1.AccessToken, t2.AccessToken)
	}
	if t1.AccessSecret != t2.AccessSecret {
		t.Fatalf("tokens access_secret mismatch: %s vs. %s", t1.AccessSecret, t2.AccessSecret)
	}
	diff := t1.AccessTokenExpiry.Sub(t2.AccessTokenExpiry)
	if diff > 1*time.Second || diff < -1*time.Second {
		t.Fatalf("tokens expiry mismatch: %s vs. %s", t1.AccessTokenExpiry, t2.AccessTokenExpiry)
	}
}

func checkErr(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

func generatePKI(t *testing.T, cn string, days int) (pemCert []byte, pemKey []byte) {
	key, cert, err := tokenpki.SelfSignedRSAKeypair(cn, days)
	if err != nil {
		t.Fatal(err)
	}
	pemCert = tokenpki.PEMCertificate(cert.Raw)
	pemKey = tokenpki.PEMRSAPrivateKey(key)
	return pemCert, pemKey
}
