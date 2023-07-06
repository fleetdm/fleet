package tables

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20230629140530(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	mdm_device_id := uuid.New().String()
	mdm_hardware_id := uuid.New().String()
	device_state := uuid.New().String()
	device_type := "CIMClient_Windows"
	device_name := "DESKTOP-1C3ARC1"
	enroll_type := "ProgrammaticEnrollment"
	enroll_user_id := ""
	enroll_proto_version := "5.0"
	enroll_client_version := "10.0.19045.2965"
	not_in_oobe := true

	insertStmt := `INSERT INTO mdm_windows_enrollments (
		mdm_device_id,
		mdm_hardware_id,
		device_state,
		device_type,
		device_name,
		enroll_type,
		enroll_user_id,
		enroll_proto_version,
		enroll_client_version,
		not_in_oobe ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(insertStmt, mdm_device_id, mdm_hardware_id, device_state, device_type, device_name, enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version, not_in_oobe)
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, mdm_device_id, mdm_hardware_id, device_state, device_type, device_name, enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version, not_in_oobe)
	require.ErrorContains(t, err, "Error 1062")

	type enrolledWindowsHost struct {
		MDMDeviceID            string    `db:"mdm_device_id"`
		MDMHardwareID          string    `db:"mdm_hardware_id"`
		MDMDeviceState         string    `db:"device_state"`
		MDMDeviceType          string    `db:"device_type"`
		MDMDeviceName          string    `db:"device_name"`
		MDMEnrollType          string    `db:"enroll_type"`
		MDMEnrollUserID        string    `db:"enroll_user_id"`
		MDMEnrollProtoVersion  string    `db:"enroll_proto_version"`
		MDMEnrollClientVersion string    `db:"enroll_client_version"`
		MDMNotInOOBE           bool      `db:"not_in_oobe"`
		CreatedAt              time.Time `db:"created_at"`
		UpdatedAt              time.Time `db:"updated_at"`
	}

	var enrolledHost enrolledWindowsHost
	selectStmt := `SELECT mdm_device_id, mdm_hardware_id, device_type, created_at, updated_at FROM mdm_windows_enrollments WHERE mdm_device_id = ?`
	err = db.Get(&enrolledHost, selectStmt, mdm_device_id)
	require.NoError(t, err)
	require.Equal(t, mdm_device_id, enrolledHost.MDMDeviceID)
	require.NotZero(t, enrolledHost.CreatedAt)
	require.NotZero(t, enrolledHost.UpdatedAt)

	_, err = db.Exec(`UPDATE mdm_windows_enrollments SET created_at = NOW()`)
	require.NoError(t, err)
}
