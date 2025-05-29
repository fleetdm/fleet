package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const serverURL = "http://androidmdm.example.com"

func TestAndroid(t *testing.T) {
	ds := CreateMySQLDS(t)
	testing_utils.TruncateTables(t, ds.primary, ds.logger, nil)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"NewAndroidHost", testNewAndroidHost},
		{"UpdateAndroidHost", testUpdateAndroidHost},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer testing_utils.TruncateTables(t, ds.primary, ds.logger, nil)
			MakeAndroidLabels(t, ds)
			c.fn(t, ds)
		})
	}
}

func testNewAndroidHost(t *testing.T, ds *Datastore) {
	const enterpriseSpecificID = "enterprise_specific_id"
	host := createAndroidHost(enterpriseSpecificID)

	result, err := ds.NewAndroidHost(testCtx(), serverURL, host)
	require.NoError(t, err)
	assert.NotZero(t, result.ID)
	assert.NotZero(t, result.Device.ID)

	lbls, err := ds.ListLabelsForHost(testCtx(), result.ID)
	require.NoError(t, err)
	require.Len(t, lbls, 2)
	names := []string{lbls[0].Name, lbls[1].Name}
	require.ElementsMatch(t, []string{fleet.BuiltinLabelNameAllHosts, fleet.BuiltinLabelNameAndroid}, names)

	resultLite, err := ds.AndroidHostLite(testCtx(), enterpriseSpecificID)
	require.NoError(t, err)
	assert.Equal(t, result.ID, resultLite.ID)
	assert.Equal(t, result.Device.ID, resultLite.Device.ID)

	// Inserting the same host again should be fine.
	// This may occur when 2 Fleet servers received the same host information via pubsub.
	resultCopy, err := ds.NewAndroidHost(testCtx(), serverURL, host)
	require.NoError(t, err)
	assert.Equal(t, result.ID, resultCopy.ID)
	assert.Equal(t, result.Device.ID, resultCopy.Device.ID)

	// create another host, this time delete the Android label
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(testCtx(), `DELETE FROM labels WHERE name = ?`, fleet.BuiltinLabelNameAndroid)
		return err
	})
	const enterpriseSpecificID2 = "enterprise_specific_id2"
	host2 := createAndroidHost(enterpriseSpecificID2)

	// still passes, but no label membership was recorded because the Android label is missing
	result, err = ds.NewAndroidHost(testCtx(), serverURL, host2)
	require.NoError(t, err)

	lbls, err = ds.ListLabelsForHost(testCtx(), result.ID)
	require.NoError(t, err)
	require.Empty(t, lbls)
}

func createAndroidHost(enterpriseSpecificID string) *android.Host {
	host := &android.Host{
		OSVersion:      "Android 14",
		Build:          "build",
		Memory:         1024,
		TeamID:         nil,
		HardwareSerial: "hardware_serial",
		CPUType:        "cpu_type",
		HardwareModel:  "hardware_model",
		HardwareVendor: "hardware_vendor",
		Device: &android.Device{
			DeviceID:             "device_id",
			EnterpriseSpecificID: ptr.String(enterpriseSpecificID),
			AndroidPolicyID:      ptr.Uint(1),
			LastPolicySyncTime:   ptr.Time(time.Now().UTC().Truncate(time.Millisecond)),
		},
	}
	host.SetNodeKey(enterpriseSpecificID)
	return host
}

func testUpdateAndroidHost(t *testing.T, ds *Datastore) {
	const enterpriseSpecificID = "es_id_update"
	host := createAndroidHost(enterpriseSpecificID)

	result, err := ds.NewAndroidHost(testCtx(), serverURL, host)
	require.NoError(t, err)
	assert.NotZero(t, result.ID)
	assert.NotZero(t, result.Device.ID)

	// Dummy update
	err = ds.UpdateAndroidHost(testCtx(), serverURL, result, false)
	require.NoError(t, err)

	host = result
	host.DetailUpdatedAt = time.Now()
	host.LabelUpdatedAt = time.Now()
	host.OSVersion = "Android 15"
	host.Build = "build_updated"
	host.Memory = 2048
	host.HardwareSerial = "hardware_serial_updated"
	host.CPUType = "cpu_type_updated"
	host.HardwareModel = "hardware_model_updated"
	host.HardwareVendor = "hardware_vendor_updated"
	host.Device.AndroidPolicyID = ptr.Uint(2)
	err = ds.UpdateAndroidHost(testCtx(), serverURL, host, false)
	require.NoError(t, err)

	resultLite, err := ds.AndroidHostLite(testCtx(), enterpriseSpecificID)
	require.NoError(t, err)
	assert.Equal(t, host.ID, resultLite.ID)
	assert.EqualValues(t, host.Device, resultLite.Device)
}

// ListLabelsForHost returns a list of fleet.Label for a given host id.
func (ds *Datastore) ListLabelsForHost(ctx context.Context, hid uint) ([]*fleet.Label, error) {
	sqlStatement := `
		SELECT labels.* from labels JOIN label_membership lm
		WHERE lm.host_id = ?
		AND lm.label_id = labels.id
	`

	labels := []*fleet.Label{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, sqlStatement, hid)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting host labels")
	}

	return labels, nil
}

func MakeAndroidLabels(t *testing.T, ds *Datastore) {
	res, err := ds.Writer(t.Context()).ExecContext(t.Context(), `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "u1", "u1@b.c",
		"1234", "salt")
	require.NoError(t, err)
	userID, _ := res.LastInsertId()

	query := `
	INSERT INTO labels (
		name,
		description,
		query,
		platform,
		label_type,
		label_membership_type,
		author_id
	) VALUES ( ?, ?, ?, ?, ?, ?, ?), ( ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = ds.Writer(t.Context()).ExecContext(
		t.Context(),
		query,
		fleet.BuiltinLabelNameAndroid,
		"",
		"",
		"android",
		fleet.LabelTypeBuiltIn,
		fleet.LabelMembershipTypeManual,
		userID,
		fleet.BuiltinLabelNameAllHosts,
		"",
		"select 1",
		"",
		fleet.LabelTypeBuiltIn,
		fleet.LabelMembershipTypeDynamic,
		userID,
	)
	require.NoError(t, err)
}
