package tables

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

type Team20220309133956 struct {
	Name   string                   `db:"name"`
	Config TeamConfig20220309133956 `db:"config"`
}

type TeamConfig20220309133956 struct {
	AgentOptions *json.RawMessage `json:"agent_options" db:"agent_options"`
}

// Scan implements the sql.Scanner interface
func (t *TeamConfig20220309133956) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		return json.Unmarshal(v, t)
	case string:
		return json.Unmarshal([]byte(v), t)
	case nil: // sql NULL
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// Value implements the sql.Valuer interface
func (t TeamConfig20220309133956) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func TestUp_20220309133956(t *testing.T) {
	db := applyUpToPrev(t)

	teams := []Team20220309133956{
		{
			Name: "test1",
		},
		{
			Name: "test2",
			Config: TeamConfig20220309133956{
				AgentOptions: ptr.RawMessage(json.RawMessage(`{"config": {"options": {"logger_plugin": "tls", "pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/v1/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}, "overrides": {}}`)),
			},
		},
	}

	_, err := db.Exec(`
INSERT INTO teams (name, agent_options)
VALUES (?, ?), (?, ?)
`, teams[0].Name, teams[0].Config.AgentOptions, teams[1].Name, teams[1].Config.AgentOptions)
	require.NoError(t, err)

	applyNext(t, db)

	var actual []Team20220309133956
	err = db.Select(&actual, `SELECT name, config from teams`)
	require.NoError(t, err)

	require.JSONEq(t, string(*teams[1].Config.AgentOptions), string(*actual[1].Config.AgentOptions))
	require.Equal(t, teams, actual)
}
