package table

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/groob/plist"
	"github.com/osquery/osquery-go/plugin/table"
)

func MDMInfo(logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("enrolled"),
		table.TextColumn("server_url"),
		table.TextColumn("checkin_url"),
		table.IntegerColumn("access_rights"),
		table.TextColumn("install_date"),
		table.TextColumn("payload_identifier"),
		table.TextColumn("topic"),
		table.TextColumn("sign_message"),
		table.TextColumn("identity_certificate_uuid"),
		table.TextColumn("has_scep_payload"),
		table.TextColumn("installed_from_dep"),
		table.TextColumn("user_approved"),
	}
	return table.NewPlugin("kolide_mdm_info", columns, generateMDMInfo)
}

func generateMDMInfo(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	profiles, err := getMDMProfile(ctx)
	if err != nil {
		return nil, err
	}

	depEnrolled, userApproved := "unknown", "unknown"
	status, err := getMDMProfileStatus(ctx)
	if err == nil { // only supported on 10.13.4+
		depEnrolled = strconv.FormatBool(status.DEPEnrolled)
		userApproved = strconv.FormatBool(status.UserApproved)
	}

	var enrollProfileItems []profileItem
	var results []map[string]string
	var mdmResults map[string]string
	for _, payload := range profiles.ComputerLevel {
		for _, item := range payload.ProfileItems {
			if item.PayloadContent == nil {
				continue
			}
			if item.PayloadType == "com.apple.mdm" {
				enrollProfile := item.PayloadContent
				enrollProfileItems = payload.ProfileItems
				mdmResults = map[string]string{
					"enrolled":                  "true",
					"server_url":                enrollProfile.ServerURL,
					"checkin_url":               enrollProfile.CheckInURL,
					"access_rights":             strconv.Itoa(enrollProfile.AccessRights),
					"install_date":              payload.ProfileInstallDate,
					"payload_identifier":        payload.ProfileIdentifier,
					"sign_message":              strconv.FormatBool(enrollProfile.SignMessage),
					"topic":                     enrollProfile.Topic,
					"identity_certificate_uuid": enrollProfile.IdentityCertificateUUID,
					"installed_from_dep":        depEnrolled,
					"user_approved":             userApproved,
				}
				break
			}
		}
	}
	if len(enrollProfileItems) != 0 {
		for _, item := range enrollProfileItems {
			if item.PayloadType == "com.apple.security.scep" {
				mdmResults["has_scep_payload"] = "true"
			}
		}
		results = append(results, mdmResults)
	} else {
		results = []map[string]string{{"enrolled": "false"}}
	}
	return results, nil
}

func getMDMProfile(ctx context.Context) (*profilesOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/bin/profiles", "-L", "-o", "stdout-xml")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("calling /usr/bin/profiles to get MDM profile payload: %w", err)
	}

	var profiles profilesOutput
	if err := plist.Unmarshal(out, &profiles); err != nil {
		return nil, fmt.Errorf("unmarshal profiles output: %w", err)
	}

	return &profiles, nil
}

type profilesOutput struct {
	ComputerLevel []profilePayload `plist:"_computerlevel"`
}

type profilePayload struct {
	ProfileIdentifier  string
	ProfileInstallDate string
	ProfileItems       []profileItem
}

type profileItem struct {
	PayloadContent *payloadContent
	PayloadType    string
}

type payloadContent struct {
	AccessRights            int
	CheckInURL              string
	ServerURL               string
	ServerCapabilities      []string
	Topic                   string
	IdentityCertificateUUID string
	SignMessage             bool
}

func getMDMProfileStatus(ctx context.Context) (profileStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/bin/profiles", "status", "-type", "enrollment")
	out, err := cmd.Output()
	if err != nil {
		return profileStatus{}, fmt.Errorf("calling /usr/bin/profiles to get MDM profile status: %w", err)
	}
	lines := bytes.Split(out, []byte("\n"))
	depEnrollmentParts := bytes.SplitN(lines[0], []byte(":"), 2)
	if len(depEnrollmentParts) < 2 {
		return profileStatus{}, fmt.Errorf("mdm: could not split the DEP Enrollment source %s", string(out))
	}
	enrollmentStatusParts := bytes.SplitN(lines[1], []byte(":"), 2)
	if len(enrollmentStatusParts) < 2 {
		return profileStatus{}, fmt.Errorf("mdm: could not split the DEP Enrollment status %s", string(out))
	}
	return profileStatus{
		DEPEnrolled:  bytes.Contains(depEnrollmentParts[1], []byte("Yes")),
		UserApproved: bytes.Contains(enrollmentStatusParts[1], []byte("Approved")),
	}, nil
}

type profileStatus struct {
	DEPEnrolled  bool
	UserApproved bool
}
