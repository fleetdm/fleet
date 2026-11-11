package service

import (
	"context"
	"crypto/x509"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) signedProfilesMatch(want, got [][]byte) {
	t := s.T()
	rootCA := x509.NewCertPool()

	assets, err := s.ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
	}, nil)
	require.NoError(t, err)

	require.True(t, rootCA.AppendCertsFromPEM(assets[fleet.MDMAssetCACert].Value))

	// verify that all the profiles were signed usign the SCEP certificate,
	// and grab their contents
	signedContents := [][]byte{}
	for _, prof := range got {
		p7, err := pkcs7.Parse(prof)
		require.NoError(t, err)
		require.NoError(t, p7.VerifyWithChain(rootCA))
		signedContents = append(signedContents, p7.Content)
	}

	// verify that contents match
	require.ElementsMatch(t, want, signedContents)
}

// those helper functions to force-set a host profile status are copied from the Datastore
// tests, couldn't put them in the test package due to circular dependency with mysql, would
// be nice to find a way to avoid this copy eventually.
func forceSetAppleHostProfileStatus(t *testing.T, ds *mysql.Datastore, hostUUID string, profile *fleet.MDMAppleConfigProfile, operation fleet.MDMOperationType, status fleet.MDMDeliveryStatus) {
	ctx := t.Context()

	// empty status string means set to NULL
	var actualStatus *fleet.MDMDeliveryStatus
	if status != "" {
		actualStatus = &status
	}

	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO host_mdm_apple_profiles
				(profile_identifier, host_uuid, status, operation_type, command_uuid, profile_name, checksum, profile_uuid)
			VALUES
				(?, ?, ?, ?, ?, ?, UNHEX(MD5(?)), ?)
			ON DUPLICATE KEY UPDATE
				status = VALUES(status),
				operation_type = VALUES(operation_type)
			`,
			profile.Identifier, hostUUID, actualStatus, operation, uuid.NewString(), profile.Name, profile.Mobileconfig, profile.ProfileUUID)
		return err
	})
}

func forceSetWindowsHostProfileStatus(t *testing.T, ds *mysql.Datastore, hostUUID string, profile *fleet.MDMWindowsConfigProfile, operation fleet.MDMOperationType, status fleet.MDMDeliveryStatus) {
	ctx := t.Context()

	// empty status string means set to NULL
	var actualStatus *fleet.MDMDeliveryStatus
	if status != "" {
		actualStatus = &status
	}

	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO host_mdm_windows_profiles
				(host_uuid, status, operation_type, command_uuid, profile_name, checksum, profile_uuid)
			VALUES
				(?, ?, ?, ?, ?, UNHEX(MD5(?)), ?)
			ON DUPLICATE KEY UPDATE
				status = VALUES(status),
				operation_type = VALUES(operation_type)
			`,
			hostUUID, actualStatus, operation, uuid.NewString(), profile.Name, profile.SyncML, profile.ProfileUUID)
		return err
	})
}

func forceSetAppleHostDeclarationStatus(t *testing.T, ds *mysql.Datastore, hostUUID string, profile *fleet.MDMAppleDeclaration, operation fleet.MDMOperationType, status fleet.MDMDeliveryStatus) {
	ctx := t.Context()

	// empty status string means set to NULL
	var actualStatus *fleet.MDMDeliveryStatus
	if status != "" {
		actualStatus = &status
	}

	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO host_mdm_apple_declarations
				(declaration_identifier, host_uuid, status, operation_type, token, declaration_name, declaration_uuid)
			VALUES
				(?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				status = VALUES(status),
				operation_type = VALUES(operation_type)
			`,
			profile.Identifier, hostUUID, actualStatus, operation, test.MakeTestBytes(), profile.Name, profile.DeclarationUUID)
		return err
	})
}

// hostHasAppleProfileOp reports whether the host has an hmap row for the given identifier
// and operation type, returning its command UUID.
func hostHasAppleProfileOp(t *testing.T, ds *mysql.Datastore, hostUUID, ident string, op fleet.MDMOperationType) (bool, string) {
	var cmdUUIDs []string
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(context.Background(), q, &cmdUUIDs,
			`SELECT command_uuid FROM host_mdm_apple_profiles
			 WHERE host_uuid = ? AND profile_identifier = ? AND operation_type = ?`,
			hostUUID, ident, op)
	})
	if len(cmdUUIDs) == 0 {
		return false, ""
	}
	require.Len(t, cmdUUIDs, 1)
	return true, cmdUUIDs[0]
}

// enrollHostDrainInitialProfiles enrolls a macOS host, delivers+acks its initial
// (fleetd/CA) profiles, and clears the reconcile-dedup key so later changes reprocess.
func (s *integrationMDMTestSuite) enrollHostDrainInitialProfiles(t *testing.T) (*fleet.Host, *mdmtest.TestAppleMDMClient) {
	ctx := t.Context()
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.awaitRunAppleMDMWorkerSchedule()
	checkNextPayloads(t, mdmDevice, false)
	require.NoError(t, s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID))
	return host, mdmDevice
}
