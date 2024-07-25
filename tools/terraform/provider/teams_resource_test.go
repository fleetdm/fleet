package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var defaultAgentOptions = `{"config":{"decorators":{"load":["SELECT uuid AS host_uuid FROM system_info;","SELECT hostname AS hostname FROM system_info;"]},"options":{"disable_distributed":false,"distributed_interval":10,"distributed_plugin":"tls","distributed_tls_max_attempts":3,"logger_tls_endpoint":"/api/osquery/log","logger_tls_period":10,"pack_delimiter":"/"}},"overrides":{}}`
var newAO = `{"command_line_flags":{"disable_events":true},"config":{"options":{"logger_tls_endpoint":"/test"}},"overrides":{}}`
var upAO = `{"command_line_flags":{"disable_events":true},"config":{"options":{"logger_tls_endpoint":"/update"}},"overrides":{}}`

// Notes:
// Set env var TF_ACC=1
// Set env var FLEETDM_APIKEY
// Set env var FLEETDM_URL (I couldn't figure out how to set this otherwise...)

// These tests use the Terraform acceptance testing framework to test
// our provider. It's a little bit magical, in that you define what
// the resource looks like in the Config and then write some Checks
// against what you actually get back. Implicit in each test is that
// it deletes the created resource at the end of the test.

func TestAccFleetdmTeams_basic(t *testing.T) {
	teamName := fmt.Sprintf("aaa-%s",
		acctest.RandStringFromCharSet(6, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name        = "%s"
						description = "Awesome description"
					}
				`, teamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", teamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", "Awesome description"),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", defaultAgentOptions),
				),
			},
		},
	})
}

func TestAccFleetdmTeams_agent_options(t *testing.T) {
	teamName := fmt.Sprintf("aaa-%s",
		acctest.RandStringFromCharSet(6, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name          = "%s"
						description   = "Awesome description"
						agent_options = jsonencode(%s)
					}
				`, teamName, newAO),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", teamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", "Awesome description"),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", newAO),
				),
			},
		},
	})
}

func TestAccFleetdmTeams_update_all(t *testing.T) {
	teamName := fmt.Sprintf("aaa-%s",
		acctest.RandStringFromCharSet(6, acctest.CharSetAlpha))
	NewTeamName := fmt.Sprintf("aaa-%s-new",
		acctest.RandStringFromCharSet(6, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name          = "%s"
						description   = "Awesome description"
					}
				`, teamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", teamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", "Awesome description"),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", defaultAgentOptions),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name          = "%s"
						description   = "Awesomer description"
						agent_options = jsonencode(%s)
					}
				`, NewTeamName, upAO),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", NewTeamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", "Awesomer description"),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", upAO),
				),
			},
		},
	})
}

func TestAccFleetdmTeams_update_each(t *testing.T) {
	teamName := fmt.Sprintf("aaa-%s",
		acctest.RandStringFromCharSet(6, acctest.CharSetAlpha))
	NewTeamName := fmt.Sprintf("aaa-%s-new",
		acctest.RandStringFromCharSet(6, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name          = "%s"
					}
				`, teamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", teamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", ""),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", defaultAgentOptions),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name          = "%s"
						description   = "Awesome description"
					}
				`, teamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", teamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", "Awesome description"),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", defaultAgentOptions),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name          = "%s"
						description   = "Awesome description"
					}
				`, teamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", teamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", "Awesome description"),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", defaultAgentOptions),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name          = "%s"
						description   = "Awesome description"
					}
				`, NewTeamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", NewTeamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", "Awesome description"),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", defaultAgentOptions),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name          = "%s"
						description   = "Awesomer description"
					}
				`, NewTeamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", NewTeamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", "Awesomer description"),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", defaultAgentOptions),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name          = "%s"
						description   = "Awesomer description"
						agent_options = jsonencode(%s)
					}
				`, NewTeamName, newAO),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", NewTeamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", "Awesomer description"),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", newAO),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "fleetdm_teams" "test_team" {
						name          = "%s"
						description   = "Awesomer description"
						agent_options = jsonencode(%s)
					}
				`, NewTeamName, upAO),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "name", NewTeamName),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "description", "Awesomer description"),
					resource.TestCheckResourceAttr("fleetdm_teams.test_team", "agent_options", upAO),
				),
			},
		},
	})
}
