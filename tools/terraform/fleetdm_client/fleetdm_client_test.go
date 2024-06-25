package fleetdm_client

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"testing"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyz")
var defaultDescription = "Awesome description"
var defaultAgentOptions = `{"config":{"decorators":{"load":["SELECT uuid AS host_uuid FROM system_info;","SELECT hostname AS hostname FROM system_info;"]},"options":{"disable_distributed":false,"distributed_interval":10,"distributed_plugin":"tls","distributed_tls_max_attempts":3,"logger_tls_endpoint":"/api/osquery/log","logger_tls_period":10,"pack_delimiter":"/"}},"overrides":{}}`

func randTeam() string {
	b := make([]rune, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return "aaa-" + string(b)
}

func TestBasic(t *testing.T) {
	apiKey := os.Getenv("FLEETDM_APIKEY")
	if apiKey == "" {
		t.Skip("FLEETDM_APIKEY not set")
	}
	apiURL := os.Getenv("FLEETDM_URL")
	if apiURL == "" {
		t.Skip("FLEETDM_URL not set")
	}
	client := NewFleetDMClient(apiURL, apiKey)
	require.NotNil(t, client)

	// Create a nice default team
	teamName := randTeam()
	team, err := client.CreateTeam(teamName, defaultDescription)
	require.NoError(t, err)
	require.NotNil(t, team)
	require.Equal(t, teamName, team.Team.Name)
	require.Equal(t, defaultDescription, team.Team.Description)
	aoBytes, err := json.Marshal(team.Team.AgentOptions)
	require.NoError(t, err)
	require.Equal(t, defaultAgentOptions, string(aoBytes))

	// And verify we can get that team back
	teamGotten, err := client.GetTeam(team.Team.ID)
	require.NoError(t, err)
	require.NotNil(t, teamGotten)
	require.Equal(t, team.Team.ID, teamGotten.Team.ID)
	require.Equal(t, team.Team.Name, teamGotten.Team.Name)
	require.Equal(t, teamName, teamGotten.Team.Name)
	require.Equal(t, defaultDescription, teamGotten.Team.Description)
	aoBytes, err = json.Marshal(teamGotten.Team.AgentOptions)
	require.NoError(t, err)
	require.Equal(t, defaultAgentOptions, string(aoBytes))

	// Set different agent options
	newAO := `{"command_line_flags":{"disable_events":true},"config":{"options":{"logger_tls_endpoint":"/test"}},"overrides":{}}`
	teamAO, err := client.UpdateAgentOptions(team.Team.ID, newAO)
	require.NoError(t, err)
	require.NotNil(t, teamAO)
	require.Equal(t, team.Team.ID, teamAO.Team.ID)
	aoBytes, err = json.Marshal(teamAO.Team.AgentOptions)
	require.NoError(t, err)
	require.Equal(t, newAO, string(aoBytes))

	// Set a different description and different name
	newName := randTeam()
	newDescription := "New description"
	teamUp, err := client.UpdateTeam(team.Team.ID, &newName, &newDescription)
	require.NoError(t, err)
	require.NotNil(t, teamUp)
	require.Equal(t, team.Team.ID, teamUp.Team.ID)
	require.Equal(t, newName, teamUp.Team.Name)
	require.Equal(t, newDescription, teamUp.Team.Description)

	// Lookup the team based on the new name
	teamId, err := client.TeamNameToID(newName)
	require.NoError(t, err)
	require.NotEqual(t, 0, teamId)
	require.Equal(t, teamUp.Team.ID, teamId)
	require.Equal(t, newName, teamUp.Team.Name)

	// And finally, delete the team
	err = client.DeleteTeam(team.Team.ID)
	require.NoError(t, err)

	// And verify it's gone
	teamDNE, err := client.GetTeam(team.Team.ID)
	require.Error(t, err)
	require.Nil(t, teamDNE)
}
