package mysql

import (
	"context"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUpdateHostCertificateTemplateStatus(t *testing.T) {
	db := CreateMySQLDS(t)
	nodeKey := uuid.New().String()
	uuid := uuid.New().String()
	hostName := "test-update-host-certificate-template"

	// Create a host
	host, err := db.NewHost(context.Background(), &fleet.Host{
		NodeKey:  &nodeKey,
		UUID:     uuid,
		Hostname: hostName,
		Platform: "android",
	})
	require.NoError(t, err)

	// TODO -- add a host certificate template when we have a foreign key set up.
	certificateTemplateID := uint(1)

	// Create a record in host_certificate_templates using ad hoc SQL
	sql := `
INSERT INTO host_certificate_templates (
	host_uuid, 
	certificate_template_id, 
	status,
	fleet_challenge
) VALUES (?, ?, ?, ?);
	`
	ExecAdhocSQL(t, db, func(q sqlx.ExtContext) error {
		_, err = q.ExecContext(context.Background(), sql, host.UUID, certificateTemplateID, "pending", "some_challenge_value")
		require.NoError(t, err)
		return nil
	})

	// Test cases
	cases := []struct {
		name             string
		templateID       uint
		newStatus        string
		expectedErrorMsg string
	}{
		{
			name:             "Valid Update",
			templateID:       certificateTemplateID,
			newStatus:        "verified",
			expectedErrorMsg: "",
		},
		{
			name:             "Invalid Status",
			templateID:       certificateTemplateID,
			newStatus:        "invalid_status",
			expectedErrorMsg: fmt.Sprintf("Invalid status '%s'", "invalid_status"),
		},
		{
			name:             "Wrong Template ID",
			templateID:       9999,
			newStatus:        "verified",
			expectedErrorMsg: fmt.Sprintf("No certificate found for host UUID '%s' and template ID '%d'", host.UUID, 9999),
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("TestUpdateHostCertificateTemplate:%s", tc.name), func(t *testing.T) {
			err := db.UpdateCertificateStatus(context.Background(), host.UUID, tc.templateID, fleet.OSSettingsStatus(tc.newStatus))
			if tc.expectedErrorMsg == "" {
				require.NoError(t, err)
				// Verify the update
				var status string
				query := `
SELECT status FROM host_certificate_templates
WHERE host_uuid = ? AND certificate_template_id = ?;
				`
				ExecAdhocSQL(t, db, func(q sqlx.ExtContext) error {
					return sqlx.GetContext(context.Background(), q, &status, query, host.UUID, tc.templateID)
				})
				require.NoError(t, err)
				require.Equal(t, tc.newStatus, status)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErrorMsg)
			}
		})
	}
}
