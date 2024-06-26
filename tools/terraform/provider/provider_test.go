package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"fleetdm": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
	apiKey := os.Getenv("FLEETDM_APIKEY")
	if apiKey == "" {
		t.Skip("FLEETDM_APIKEY not set")
	}

	// I don't like this, but I can't figure out how to pass the url otherwise...
	url := os.Getenv("FLEETDM_URL")
	if url == "" {
		t.Skip("FLEETDM_URL not set")
	}
}
