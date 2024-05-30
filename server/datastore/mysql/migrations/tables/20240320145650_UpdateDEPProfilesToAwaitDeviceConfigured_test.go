package tables

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20240320145650(t *testing.T) {
	db := applyUpToPrev(t)

	type macosSetupAssistantArgs struct {
		Task              string   `json:"task"`
		TeamID            *uint    `json:"team_id,omitempty"`
		HostSerialNumbers []string `json:"host_serial_numbers,omitempty"`
	}

	type job struct {
		ID        uint             `json:"id" db:"id"`
		CreatedAt time.Time        `json:"created_at" db:"created_at"`
		UpdatedAt *time.Time       `json:"updated_at" db:"updated_at"`
		Name      string           `json:"name" db:"name"`
		Args      *json.RawMessage `json:"args" db:"args"`
		State     string           `json:"state" db:"state"`
		Retries   int              `json:"retries" db:"retries"`
		Error     string           `json:"error" db:"error"`
		NotBefore time.Time        `json:"not_before" db:"not_before"`
	}

	var jobs []*job
	err := db.Select(&jobs, `SELECT id, name, args, state, retries, error, not_before FROM jobs`)
	require.NoError(t, err)
	require.Empty(t, jobs)

	applyNext(t, db)

	err = db.Select(&jobs, `SELECT id, name, args, state, retries, error, not_before FROM jobs`)
	require.NoError(t, err)
	require.Len(t, jobs, 1)

	require.Equal(t, "macos_setup_assistant", jobs[0].Name)
	require.Equal(t, 0, jobs[0].Retries)
	require.LessOrEqual(t, jobs[0].NotBefore, time.Now().UTC())
	require.NotNil(t, jobs[0].Args)

	var args macosSetupAssistantArgs
	err = json.Unmarshal(*jobs[0].Args, &args)
	require.NoError(t, err)
	require.Equal(t, "update_all_profiles", args.Task)
}
