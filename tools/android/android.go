package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/option"
)

// Required env vars:
var (
	androidServiceCredentials string
	androidProjectID          string
)

const (
	cmdEnterprisesDelete          = "enterprises.delete"
	cmdEnterprisesList            = "enterprises.list"
	cmdEnterprisesWebTokensCreate = "enterprises.webTokens.create"
	cmdApplicationsGet            = "applications.get"
	cmdPoliciesList               = "policies.list"
	cmdPoliciesDelete             = "policies.delete"
	cmdDevicesList                = "devices.list"
	cmdDevicesDelete              = "devices.delete"
	cmdDevicesRelinquish          = "devices.issueCommand.RELINQUISH_OWNERSHIP"
)

var commands = []string{
	cmdEnterprisesDelete,
	cmdEnterprisesList,
	cmdEnterprisesWebTokensCreate,
	cmdApplicationsGet,
	cmdPoliciesList,
	cmdPoliciesDelete,
	cmdDevicesList,
	cmdDevicesDelete,
	cmdDevicesRelinquish,
}

func main() {
	dev_mode.IsEnabled = true
	androidServiceCredentials = dev_mode.Env("FLEET_DEV_ANDROID_GOOGLE_SERVICE_CREDENTIALS")
	if androidServiceCredentials == "" {
		log.Fatal("FLEET_DEV_ANDROID_GOOGLE_SERVICE_CREDENTIALS must be set")
	}

	type credentials struct {
		ProjectID string `json:"project_id"`
	}
	var creds credentials
	err := json.Unmarshal([]byte(androidServiceCredentials), &creds)
	if err != nil {
		log.Fatalf("unmarshaling android service credentials: %s", err)
	}
	androidProjectID = creds.ProjectID
	if androidProjectID == "" {
		log.Fatal("project_id not found in android service credentials")
	}

	command := flag.String("command", "", strings.Join(commands, "\n"))
	enterpriseID := flag.String("enterprise_id", "", "")
	deviceID := flag.String("device_id", "", "")
	policyID := flag.String("policy_id", "", "")
	flag.Parse()

	if !slices.Contains(commands, *command) {
		flag.Usage()
		os.Exit(1)
	}

	// Normalize enterprise_id by stripping "enterprises/" prefix if present
	if *enterpriseID != "" {
		*enterpriseID = strings.TrimPrefix(*enterpriseID, "enterprises/")
	}

	if slices.Index(commands, *command) == -1 {
		log.Fatalf("Command must be one of: %s", strings.Join(commands, ", "))
	}

	ctx := context.Background()
	mgmt, err := androidmanagement.NewService(ctx, option.WithCredentialsJSON([]byte(androidServiceCredentials)))
	if err != nil {
		log.Fatalf("Error creating android management service: %v", err)
	}

	switch *command {
	case cmdEnterprisesDelete:
		enterprisesDelete(mgmt, *enterpriseID)
	case cmdEnterprisesList:
		enterprisesList(mgmt)
	case cmdEnterprisesWebTokensCreate:
		enterprisesWebTokensCreate(mgmt, *enterpriseID)
	case cmdApplicationsGet:
		applicationsGet(mgmt, *enterpriseID, flag.Arg(0))
	case cmdPoliciesList:
		policiesList(mgmt, *enterpriseID)
	case cmdPoliciesDelete:
		policiesDelete(mgmt, *enterpriseID, *policyID)
	case cmdDevicesList:
		devicesList(mgmt, *enterpriseID)
	case cmdDevicesDelete:
		devicesDelete(mgmt, *enterpriseID, *deviceID)
	case cmdDevicesRelinquish:
		devicesRelinquishOwnership(mgmt, *enterpriseID, *deviceID)
	default:
		log.Fatalf("Unknown command: %s", *command)
	}

}

func enterprisesDelete(mgmt *androidmanagement.Service, enterpriseID string) {
	if enterpriseID == "" {
		log.Fatalf("enterprise_id must be set")
	}
	_, err := mgmt.Enterprises.Delete("enterprises/" + enterpriseID).Do()
	if err != nil {
		log.Fatalf("Error deleting enterprise: %v", err)
	}
}

func enterprisesList(mgmt *androidmanagement.Service) {
	ctx := context.Background()
	var enterprises []*androidmanagement.Enterprise
	var callCount int
	err := mgmt.Enterprises.List().ProjectId(androidProjectID).Pages(ctx, func(page *androidmanagement.ListEnterprisesResponse) error {
		callCount++
		enterprises = append(enterprises, page.Enterprises...)
		return nil
	})
	if err != nil {
		log.Fatalf("Error listing enterprises: %v", err)
	}
	if len(enterprises) == 0 {
		log.Printf("No enterprises found")
		return
	}
	for _, enterprise := range enterprises {
		log.Printf("Enterprise: %+v", *enterprise)
	}
	log.Printf("%d enterprises found in %d pages", len(enterprises), callCount)
}

func policiesList(mgmt *androidmanagement.Service, enterpriseID string) {
	if enterpriseID == "" {
		log.Fatalf("enterprise_id must be set")
	}
	result, err := mgmt.Enterprises.Policies.List("enterprises/" + enterpriseID).Do()
	if err != nil {
		log.Fatalf("Error listing policies: %v", err)
	}
	if len(result.Policies) == 0 {
		log.Printf("No policies found")
		return
	}
	b, err := json.Marshal(result.Policies, jsontext.WithIndent("  "))
	if err != nil {
		log.Fatalf("Error marshalling policies: %v", err)
	}
	fmt.Println(string(b))
}

func applicationsGet(mgmt *androidmanagement.Service, enterpriseID string, applicationID string) {
	if enterpriseID == "" {
		log.Fatalf("enterprise_id must be set")
	}
	if applicationID == "" {
		log.Fatal("application ID argument missing")
	}
	result, err := mgmt.Enterprises.Applications.Get(fmt.Sprintf("enterprises/%s/applications/%s", enterpriseID, applicationID)).Do()
	if err != nil {
		log.Fatalf("Error getting application: %v", err)
	}
	b, err := json.Marshal(result, jsontext.WithIndent("  "))
	if err != nil {
		log.Fatalf("Error marshalling application: %v", err)
	}
	fmt.Println(string(b))
}

func policiesDelete(mgmt *androidmanagement.Service, enterpriseID, policyID string) {
	if enterpriseID == "" || policyID == "" {
		log.Fatalf("enterprise_id and policy_id must be set")
	}
	_, err := mgmt.Enterprises.Policies.Delete("enterprises/" + enterpriseID + "/policies/" + policyID).Do()
	if err != nil {
		log.Fatalf("Error deleting policy: %v", err)
	}
	log.Printf("Policy %s deleted", policyID)
}

func devicesList(mgmt *androidmanagement.Service, enterpriseID string) {
	if enterpriseID == "" {
		log.Fatalf("enterprise_id must be set")
	}
	result, err := mgmt.Enterprises.Devices.List("enterprises/" + enterpriseID).Do()
	if err != nil {
		log.Fatalf("Error listing devices: %v", err)
	}
	if len(result.Devices) == 0 {
		log.Printf("No policies found")
		return
	}
	b, err := json.Marshal(result.Devices, jsontext.WithIndent("  "))
	if err != nil {
		log.Fatalf("Error marshalling devices: %v", err)
	}
	fmt.Println(string(b))
	log.Printf("Total devices: %d", len(result.Devices))
}

func devicesDelete(mgmt *androidmanagement.Service, enterpriseID string, deviceID string) {
	if enterpriseID == "" || deviceID == "" {
		log.Fatalf("enterprise_id and device_id must be set")
	}
	_, err := mgmt.Enterprises.Devices.Delete("enterprises/" + enterpriseID + "/devices/" + deviceID).Do()
	if err != nil {
		log.Fatalf("Error listing devices: %v", err)
	}
	log.Printf("Device %s deleted", deviceID)
}

func devicesRelinquishOwnership(mgmt *androidmanagement.Service, enterpriseID, deviceID string) {
	if enterpriseID == "" || deviceID == "" {
		log.Fatalf("enterprise_id and device_id must be set")
	}
	operation, err := mgmt.Enterprises.Devices.IssueCommand("enterprises/"+enterpriseID+"/devices/"+deviceID, &androidmanagement.Command{
		Type: "RELINQUISH_OWNERSHIP",
	}).Do()
	if err != nil {
		log.Fatalf("Error issuing command: %v", err)
	}
	data, err := json.Marshal(operation, jsontext.WithIndent("  "))
	if err != nil {
		log.Fatalf("Error marshalling operation: %v", err)
	}
	log.Println(string(data))
}

func enterprisesWebTokensCreate(mgmt *androidmanagement.Service, enterpriseID string) {
	if enterpriseID == "" {
		log.Fatalf("enterprise_id must be set")
	}

	webToken := &androidmanagement.WebToken{
		ParentFrameUrl: "https://example.com",
		Permissions:    []string{"APPROVE_APPS"},
	}

	result, err := mgmt.Enterprises.WebTokens.Create("enterprises/"+enterpriseID, webToken).Do()
	if err != nil {
		log.Fatalf("Error creating web token: %v", err)
	}

	data, err := json.Marshal(result, jsontext.WithIndent("  "))
	if err != nil {
		log.Fatalf("Error marshalling web token: %v", err)
	}
	log.Println(string(data))

	// Construct and display the complete URL
	if result.Value != "" {
		playURL := "https://play.google.com/work/embedded/search?token=" + result.Value
		log.Printf("\nComplete Play Store URL:\n%s\n", playURL)
	}
}
