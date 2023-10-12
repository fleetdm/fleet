package service

import (
	"github.com/fleetdm/fleet/v4/server/ptr"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOptionsFromRequest(t *testing.T) {
	var listOptionsTests = []struct {
		// url string to parse
		url string
		// expected list options
		listOptions fleet.ListOptions
		// should cause a BadRequest error
		shouldErr400 bool
	}{
		// both params provided
		{
			url:         "/foo?page=1&per_page=10",
			listOptions: fleet.ListOptions{Page: 1, PerPage: 10},
		},
		// only per_page (page should default to 0)
		{
			url:         "/foo?per_page=10",
			listOptions: fleet.ListOptions{Page: 0, PerPage: 10},
		},
		// only page (per_page should default to defaultPerPage
		{
			url:         "/foo?page=10",
			listOptions: fleet.ListOptions{Page: 10, PerPage: defaultPerPage},
		},
		// no params provided (defaults to empty ListOptions indicating
		// unlimited)
		{
			url:         "/foo?unrelated=foo",
			listOptions: fleet.ListOptions{},
		},

		// Both order params provided
		{
			url:         "/foo?order_key=foo&order_direction=desc",
			listOptions: fleet.ListOptions{OrderKey: "foo", OrderDirection: fleet.OrderDescending},
		},
		// Both order params provided (asc)
		{
			url:         "/foo?order_key=bar&order_direction=asc",
			listOptions: fleet.ListOptions{OrderKey: "bar", OrderDirection: fleet.OrderAscending},
		},
		// Default order direction
		{
			url:         "/foo?order_key=foo",
			listOptions: fleet.ListOptions{OrderKey: "foo", OrderDirection: fleet.OrderAscending},
		},

		// All params defined
		{
			url: "/foo?order_key=foo&order_direction=desc&page=1&per_page=100",
			listOptions: fleet.ListOptions{
				OrderKey:       "foo",
				OrderDirection: fleet.OrderDescending,
				Page:           1,
				PerPage:        100,
			},
		},

		// various 400 error cases
		{
			url:          "/foo?page=foo&per_page=10",
			shouldErr400: true,
		},
		{
			url:          "/foo?page=1&per_page=foo",
			shouldErr400: true,
		},
		{
			url:          "/foo?page=-1",
			shouldErr400: true,
		},
		{
			url:          "/foo?page=-1&per_page=-10",
			shouldErr400: true,
		},
		{
			url:          "/foo?page=1&order_direction=desc",
			shouldErr400: true,
		},
		{
			url:          "/foo?&order_direction=foo&order_key=",
			shouldErr400: true,
		},
	}

	for _, tt := range listOptionsTests {
		t.Run(
			tt.url, func(t *testing.T) {
				urlStruct, _ := url.Parse(tt.url)
				req := &http.Request{URL: urlStruct}
				opt, err := listOptionsFromRequest(req)

				if tt.shouldErr400 {
					assert.NotNil(t, err)
					var be *fleet.BadRequestError
					require.ErrorAs(t, err, &be)
					return
				}

				assert.Nil(t, err)
				assert.Equal(t, tt.listOptions, opt)

			},
		)
	}
}

func TestHostListOptionsFromRequest(t *testing.T) {
	var hostListOptionsTests = map[string]struct {
		// url string to parse
		url string
		// expected options
		hostListOptions fleet.HostListOptions
		// expected error message, if any
		errorMessage string
	}{
		"no params passed": {
			url:             "/foo",
			hostListOptions: fleet.HostListOptions{},
		},
		"embedded list options params defined": {
			url: "/foo?order_key=foo&order_direction=desc&page=1&per_page=100",
			hostListOptions: fleet.HostListOptions{
				ListOptions: fleet.ListOptions{
					OrderKey:       "foo",
					OrderDirection: fleet.OrderDescending,
					Page:           1,
					PerPage:        100,
				},
			},
		},
		"all params defined": {
			url: "/foo?order_key=foo&order_direction=asc&page=10&per_page=1&device_mapping=T&additional_info_filters" +
				"=filter1,filter2&status=new&team_id=2&policy_id=3&policy_response=passing&software_id=4&os_id=5" +
				"&os_name=osName&os_version=osVersion&disable_failing_policies=1&macos_settings=verified" +
				"&macos_settings_disk_encryption=enforcing&os_settings=pending&os_settings_disk_encryption=failed" +
				"&bootstrap_package=installed&mdm_id=6&mdm_name=mdmName&mdm_enrollment_status=automatic" +
				"&munki_issue_id=7&low_disk_space=99",
			hostListOptions: fleet.HostListOptions{
				ListOptions: fleet.ListOptions{
					OrderKey:       "foo",
					OrderDirection: fleet.OrderAscending,
					Page:           10,
					PerPage:        1,
				},
				DeviceMapping:                     true,
				AdditionalFilters:                 []string{"filter1", "filter2"},
				StatusFilter:                      fleet.StatusNew,
				TeamFilter:                        ptr.Uint(2),
				PolicyIDFilter:                    ptr.Uint(3),
				PolicyResponseFilter:              ptr.Bool(true),
				SoftwareIDFilter:                  ptr.Uint(4),
				OSIDFilter:                        ptr.Uint(5),
				OSNameFilter:                      ptr.String("osName"),
				OSVersionFilter:                   ptr.String("osVersion"),
				DisableFailingPolicies:            true,
				MacOSSettingsFilter:               fleet.OSSettingsVerified,
				MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing,
				OSSettingsFilter:                  fleet.OSSettingsPending,
				OSSettingsDiskEncryptionFilter:    fleet.DiskEncryptionFailed,
				MDMBootstrapPackageFilter:         (*fleet.MDMBootstrapPackageStatus)(ptr.String(string(fleet.MDMBootstrapPackageInstalled))),
				MDMIDFilter:                       ptr.Uint(6),
				MDMNameFilter:                     ptr.String("mdmName"),
				MDMEnrollmentStatusFilter:         fleet.MDMEnrollStatusAutomatic,
				MunkiIssueIDFilter:                ptr.Uint(7),
				LowDiskSpaceFilter:                ptr.Int(99),
			},
		},
		"error in page (embedded list options)": {
			url:          "/foo?page=-1",
			errorMessage: "negative page value",
		},
		"error in status": {
			url:          "/foo?status=foo",
			errorMessage: "Invalid status",
		},
		"error in team_id (number too large)": {
			url:          "/foo?team_id=9,223,372,036,854,775,808",
			errorMessage: "Invalid team_id",
		},
		"error in team_id (not a number)": {
			url:          "/foo?team_id=foo",
			errorMessage: "Invalid team_id",
		},
		"error in policy_id": {
			url:          "/foo?policy_id=foo",
			errorMessage: "Invalid policy_id",
		},
		"error when policy_response specified without policy_id": {
			url:          "/foo?policy_response=passing",
			errorMessage: "Missing policy_id",
		},
		"error in policy_response (invalid option)": {
			url:          "/foo?policy_id=1&policy_response=foo",
			errorMessage: "Invalid policy_response",
		},
		"error in software_id": {
			url:          "/foo?software_id=foo",
			errorMessage: "Invalid software_id",
		},
		"error in od_id": {
			url:          "/foo?os_id=foo",
			errorMessage: "Invalid os_id",
		},
		"error in disable_failing_policies": {
			url:          "/foo?disable_failing_policies=foo",
			errorMessage: "Invalid disable_failing_policies",
		},
		"error in device_mapping": {
			url:          "/foo?device_mapping=foo",
			errorMessage: "Invalid device_mapping",
		},
		"error in mdm_id": {
			url:          "/foo?mdm_id=foo",
			errorMessage: "Invalid mdm_id",
		},
		"error in mdm_enrollment_status (invalid option)": {
			url:          "/foo?mdm_enrollment_status=foo",
			errorMessage: "Invalid mdm_enrollment_status",
		},
		"error in macos_settings (invalid option)": {
			url:          "/foo?macos_settings=foo",
			errorMessage: "Invalid macos_settings",
		},
		"error in macos_settings_disk_encryption (invalid option)": {
			url:          "/foo?macos_settings_disk_encryption=foo",
			errorMessage: "Invalid macos_settings_disk_encryption",
		},
		"error in os_settings (invalid option)": {
			url:          "/foo?os_settings=foo",
			errorMessage: "Invalid os_settings",
		},
		"error in os_settings_disk_encryption (invalid option)": {
			url:          "/foo?os_settings_disk_encryption=foo",
			errorMessage: "Invalid os_settings_disk_encryption",
		},
		"error in bootstrap_package (invalid option)": {
			url:          "/foo?bootstrap_package=foo",
			errorMessage: "Invalid bootstrap_package",
		},
		"error in munki_issue_id": {
			url:          "/foo?munki_issue_id=foo",
			errorMessage: "Invalid munki_issue_id",
		},
		// error in low_disk_space
		"error in low_disk_space (not a number)": {
			url:          "/foo?low_disk_space=foo",
			errorMessage: "Invalid low_disk_space",
		},
		"error in low_disk_space (too low)": {
			url:          "/foo?low_disk_space=0",
			errorMessage: "Invalid low_disk_space",
		},
		"error in low_disk_space (too high)": {
			url:          "/foo?low_disk_space=101",
			errorMessage: "Invalid low_disk_space",
		},
	}

	for name, tt := range hostListOptionsTests {
		t.Run(
			name, func(t *testing.T) {
				urlStruct, _ := url.Parse(tt.url)
				req := &http.Request{URL: urlStruct}
				opt, err := hostListOptionsFromRequest(req)

				if tt.errorMessage != "" {
					assert.NotNil(t, err)
					var be *fleet.BadRequestError
					require.ErrorAs(t, err, &be)
					assert.True(
						t, strings.Contains(err.Error(), tt.errorMessage),
						"error message '%v' should contain '%v'", err.Error(), tt.errorMessage,
					)
					return
				}

				assert.Nil(t, err)
				assert.Equal(t, tt.hostListOptions, opt)

			},
		)
	}
}
