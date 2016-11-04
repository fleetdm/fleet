package service

import (
	"testing"

	"github.com/kolide/kolide-ose/server/datastore"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestCreateOrgInfo(t *testing.T) {
	ds, err := datastore.New("inmem", "")
	require.Nil(t, err)
	svc, err := newTestService(ds)
	require.Nil(t, err)
	var orgInfoTests = []struct {
		infoPayload kolide.AppConfigPayload
	}{
		{
			infoPayload: kolide.AppConfigPayload{
				OrgInfo: &kolide.OrgInfo{
					OrgLogoURL: stringPtr("acme.co/images/logo.png"),
					OrgName:    stringPtr("Acme"),
				},
				ServerSettings: &kolide.ServerSettings{
					KolideServerURL: stringPtr("https://acme.co:8080/"),
				},
			},
		},
	}

	for _, tt := range orgInfoTests {
		result, err := svc.NewAppConfig(context.Background(), tt.infoPayload)
		require.Nil(t, err)

		payload := tt.infoPayload
		assert.NotEmpty(t, result.ID)
		assert.Equal(t, *payload.OrgInfo.OrgLogoURL, result.OrgLogoURL)
		assert.Equal(t, *payload.OrgInfo.OrgName, result.OrgName)
		assert.Equal(t, *payload.ServerSettings.KolideServerURL, result.KolideServerURL)
	}
}
