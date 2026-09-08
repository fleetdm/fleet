package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestAppleBM(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SetABMTokenDefault", testSetABMTokenDefault},
		{"ClearABMTokenDefault", testClearABMTokenDefault},
		{"SetABMTokenServerUUID", testSetABMTokenServerUUID},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func insertTestABMToken(t *testing.T, ds *Datastore, orgName string) *fleet.ABMToken {
	tok, err := ds.InsertABMToken(t.Context(), &fleet.ABMToken{
		OrganizationName: orgName,
		EncryptedToken:   []byte(uuid.NewString()),
		RenewAt:          time.Now().Add(365 * 24 * time.Hour),
	})
	require.NoError(t, err)
	return tok
}

// getTestABMTokenRow reads the raw columns of an ABM token, so that tests can
// distinguish a NULL server_uuid from an empty string.
func getTestABMTokenRow(t *testing.T, ds *Datastore, tokenID uint) (isDefault bool, serverUUID *string) {
	var row struct {
		IsDefault  bool    `db:"is_default"`
		ServerUUID *string `db:"server_uuid"`
	}
	err := sqlx.GetContext(context.Background(), ds.writer(context.Background()), &row,
		"SELECT is_default, server_uuid FROM abm_tokens WHERE id = ?", tokenID)
	require.NoError(t, err)
	return row.IsDefault, row.ServerUUID
}

func testSetABMTokenDefault(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// no such token
	err := ds.SetABMTokenDefault(ctx, 999)
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// the first token inserted is default on insert
	tok1 := insertTestABMToken(t, ds, "org1")
	require.True(t, tok1.IsDefault)
	isDefault, _ := getTestABMTokenRow(t, ds, tok1.ID)
	require.True(t, isDefault)

	// a second token is not default
	tok2 := insertTestABMToken(t, ds, "org2")
	require.False(t, tok2.IsDefault)
	isDefault, _ = getTestABMTokenRow(t, ds, tok2.ID)
	require.False(t, isDefault)

	// setting the default is exclusive: tok1 gets cleared
	require.NoError(t, ds.SetABMTokenDefault(ctx, tok2.ID))
	isDefault, _ = getTestABMTokenRow(t, ds, tok1.ID)
	require.False(t, isDefault)
	isDefault, _ = getTestABMTokenRow(t, ds, tok2.ID)
	require.True(t, isDefault)

	// setting the same token again is a no-op
	require.NoError(t, ds.SetABMTokenDefault(ctx, tok2.ID))
	isDefault, _ = getTestABMTokenRow(t, ds, tok2.ID)
	require.True(t, isDefault)
	isDefault, _ = getTestABMTokenRow(t, ds, tok1.ID)
	require.False(t, isDefault)

	// a not-found token leaves the existing default untouched
	err = ds.SetABMTokenDefault(ctx, tok2.ID+1000)
	require.Error(t, err)
	require.ErrorAs(t, err, &nfe)
	isDefault, _ = getTestABMTokenRow(t, ds, tok2.ID)
	require.True(t, isDefault)
}

func testClearABMTokenDefault(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// no tokens at all
	require.NoError(t, ds.ClearABMTokenDefault(ctx))

	// a sole token stays default
	tok1 := insertTestABMToken(t, ds, "org1")
	require.NoError(t, ds.ClearABMTokenDefault(ctx))
	isDefault, _ := getTestABMTokenRow(t, ds, tok1.ID)
	require.True(t, isDefault)

	// with two tokens, the default is cleared
	tok2 := insertTestABMToken(t, ds, "org2")
	require.NoError(t, ds.ClearABMTokenDefault(ctx))
	isDefault, _ = getTestABMTokenRow(t, ds, tok1.ID)
	require.False(t, isDefault)
	isDefault, _ = getTestABMTokenRow(t, ds, tok2.ID)
	require.False(t, isDefault)

	// clearing again is a no-op
	require.NoError(t, ds.ClearABMTokenDefault(ctx))
	tokens, err := ds.ListABMTokens(ctx)
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	for _, tok := range tokens {
		require.False(t, tok.IsDefault, tok.OrganizationName)
	}

	// clearing works when the default was set explicitly
	require.NoError(t, ds.SetABMTokenDefault(ctx, tok2.ID))
	require.NoError(t, ds.ClearABMTokenDefault(ctx))
	isDefault, _ = getTestABMTokenRow(t, ds, tok2.ID)
	require.False(t, isDefault)
}

func testSetABMTokenServerUUID(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// no such token
	err := ds.SetABMTokenServerUUID(ctx, 999, "some-uuid")
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	tok1 := insertTestABMToken(t, ds, "org1")
	tok2 := insertTestABMToken(t, ds, "org2")

	// server_uuid is NULL until set
	_, serverUUID := getTestABMTokenRow(t, ds, tok1.ID)
	require.Nil(t, serverUUID)

	// setting it only affects the requested token
	require.NoError(t, ds.SetABMTokenServerUUID(ctx, tok1.ID, "server-uuid-1"))
	_, serverUUID = getTestABMTokenRow(t, ds, tok1.ID)
	require.NotNil(t, serverUUID)
	require.Equal(t, "server-uuid-1", *serverUUID)
	_, serverUUID = getTestABMTokenRow(t, ds, tok2.ID)
	require.Nil(t, serverUUID)

	// overwriting an existing value
	require.NoError(t, ds.SetABMTokenServerUUID(ctx, tok1.ID, "server-uuid-1-updated"))
	_, serverUUID = getTestABMTokenRow(t, ds, tok1.ID)
	require.NotNil(t, serverUUID)
	require.Equal(t, "server-uuid-1-updated", *serverUUID)

	// an empty value is stored as-is
	require.NoError(t, ds.SetABMTokenServerUUID(ctx, tok1.ID, ""))
	_, serverUUID = getTestABMTokenRow(t, ds, tok1.ID)
	require.NotNil(t, serverUUID)
	require.Empty(t, *serverUUID)

	// setting the default flag is unaffected by the server UUID update
	isDefault, _ := getTestABMTokenRow(t, ds, tok1.ID)
	require.True(t, isDefault)

	// values are surfaced through ListABMTokens, with NULL coalesced to ""
	require.NoError(t, ds.SetABMTokenServerUUID(ctx, tok1.ID, "server-uuid-1"))
	tokens, err := ds.ListABMTokens(ctx)
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	byOrg := make(map[string]string, len(tokens))
	for _, tok := range tokens {
		byOrg[tok.OrganizationName] = tok.ServerUUID
	}
	require.Equal(t, map[string]string{"org1": "server-uuid-1", "org2": ""}, byOrg)
}
