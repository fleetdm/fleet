package test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func CreateVPPTokenData(expiration time.Time, orgName, location string) (*fleet.VPPTokenData, error) {
	var randBytes [32]byte
	_, err := rand.Read(randBytes[:])
	if err != nil {
		return nil, fmt.Errorf("generating random bytes: %w", err)
	}
	token := base64.StdEncoding.EncodeToString(randBytes[:])
	raw := fleet.VPPTokenRaw{
		OrgName: orgName,
		Token:   token,
		ExpDate: expiration.Format("2006-01-02T15:04:05Z0700"),
	}
	rawJson, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshalling vpp raw token: %w", err)
	}

	base64Token := base64.StdEncoding.EncodeToString(rawJson)
	return &fleet.VPPTokenData{Token: base64Token, Location: location}, nil
}

func CreateInsertGlobalVPPToken(t *testing.T, ds fleet.Datastore) *fleet.VPPTokenDB {
	ctx := context.Background()
	dataToken, err := CreateVPPTokenData(time.Now().Add(24*time.Hour), "Donkey Kong", "Jungle")
	require.NoError(t, err)
	tok1, err := ds.InsertVPPToken(ctx, dataToken)
	assert.NoError(t, err)
	tok1New, err := ds.UpdateVPPTokenTeams(ctx, tok1.ID, []uint{})
	assert.NoError(t, err)

	return tok1New
}

func CreateVPPTokenEncoded(expiration time.Time, orgName, location string) ([]byte, error) {
	dataToken, err := CreateVPPTokenData(expiration, orgName, location)
	if err != nil {
		return nil, err
	}
	return []byte(dataToken.Token), nil
}

func CreateVPPTokenEncodedAfterMigration(expiration time.Time, orgName, location string) ([]byte, error) {
	dataToken, err := CreateVPPTokenData(expiration, orgName, location)
	if err != nil {
		return nil, err
	}

	dataTokenJson, err := json.Marshal(dataToken)
	if err != nil {
		return nil, fmt.Errorf("marshalling vpp data token: %w", err)
	}
	return dataTokenJson, nil
}
