package service

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOptionsFromRequest(t *testing.T) {
	listOptionsTests := []struct {
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
			url: "/foo?order_key=foo&order_direction=desc&page=1&per_page=100&after=bar",
			listOptions: fleet.ListOptions{
				OrderKey:       "foo",
				OrderDirection: fleet.OrderDescending,
				Page:           1,
				PerPage:        100,
				After:          "bar",
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
			url:          "/foo?page=1&per_page=-10",
			shouldErr400: true,
		},
		// order_direction without order_key
		{
			url:          "/foo?page=1&order_direction=desc",
			shouldErr400: true,
		},
		// bad order_direction
		{
			url:          "/foo?&order_direction=foo&order_key=foo",
			shouldErr400: true,
		},
		// after without order_key
		{
			url:          "/foo?page=1&after=foo",
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
	hostListOptionsTests := map[string]struct {
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
				"&os_name=osName&os_version=osVersion&os_version_id=5&disable_failing_policies=1&macos_settings=verified" +
				"&macos_settings_disk_encryption=enforcing&os_settings=pending&os_settings_disk_encryption=failed" +
				"&bootstrap_package=installed&mdm_id=6&mdm_name=mdmName&mdm_enrollment_status=automatic" +
				"&munki_issue_id=7&low_disk_space=99&vulnerability=CVE-2023-42887",
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
				OSVersionIDFilter:                 ptr.Uint(5),
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
				VulnerabilityFilter:               ptr.String("CVE-2023-42887"),
			},
		},
		"policy_id and policy_response params (for coverage)": {
			url: "/foo?policy_id=100&policy_response=failing",
			hostListOptions: fleet.HostListOptions{
				PolicyIDFilter:       ptr.Uint(100),
				PolicyResponseFilter: ptr.Bool(false),
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
		"error in os_id": {
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
		"error in os_name/os_version (os_name missing)": {
			url:          "/foo?os_version=1.0",
			errorMessage: "Invalid os_name",
		},
		"error in os_name/os_version (os_version missing)": {
			url:          "/foo?os_name=foo",
			errorMessage: "Invalid os_version",
		},
		"negative software_id": {
			url:          "/foo?software_id=-10",
			errorMessage: "Invalid software_id",
		},
		"negative software_version_id": {
			url:          "/foo?software_version_id=-10",
			errorMessage: "Invalid software_version_id",
		},
		"negative software_title_id": {
			url:          "/foo?software_title_id=-10",
			errorMessage: "Invalid software_title_id",
		},
		"software_title_id too big": {
			url:          "/foo?software_title_id=" + fmt.Sprint(1<<33),
			errorMessage: "Invalid software_title_id",
		},
		"software_version_id can be > 32bits": {
			url: "/foo?software_version_id=" + fmt.Sprint(1<<33),
			hostListOptions: fleet.HostListOptions{
				SoftwareVersionIDFilter: ptr.Uint(1 << 33),
			},
		},
		"good software_version_id": {
			url: "/foo?software_version_id=1",
			hostListOptions: fleet.HostListOptions{
				SoftwareVersionIDFilter: ptr.Uint(1),
			},
		},
		"good software_title_id": {
			url: "/foo?software_title_id=1",
			hostListOptions: fleet.HostListOptions{
				SoftwareTitleIDFilter: ptr.Uint(1),
			},
		},
		"invalid combination software_title_id and software_version_id": {
			url:          "/foo?software_title_id=1&software_version_id=2",
			errorMessage: "The combination of software_version_id and software_title_id is not allowed",
		},
		"invalid combination software_id and software_version_id": {
			url:          "/foo?software_id=1&software_version_id=2",
			errorMessage: "The combination of software_id and software_version_id is not allowed",
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
					require.Contains(t, err.Error(), tt.errorMessage)
					return
				}
				assert.Nil(t, err)
				assert.Equal(t, tt.hostListOptions, opt)
			},
		)
	}
}

func TestCarveListOptionsFromRequest(t *testing.T) {
	carveListOptionsTests := map[string]struct {
		// url string to parse
		url string
		// expected options
		carveListOptions fleet.CarveListOptions
		// expected error message, if any
		errorMessage string
	}{
		"no params passed": {
			url:              "/foo",
			carveListOptions: fleet.CarveListOptions{},
		},
		"embedded list options params defined": {
			url: "/foo?order_key=foo&order_direction=desc&page=1&per_page=100",
			carveListOptions: fleet.CarveListOptions{
				ListOptions: fleet.ListOptions{
					OrderKey:       "foo",
					OrderDirection: fleet.OrderDescending,
					Page:           1,
					PerPage:        100,
				},
			},
		},
		"all params defined": {
			url: "/foo?order_key=foo&order_direction=asc&page=10&per_page=1&expired=true",
			carveListOptions: fleet.CarveListOptions{
				ListOptions: fleet.ListOptions{
					OrderKey:       "foo",
					OrderDirection: fleet.OrderAscending,
					Page:           10,
					PerPage:        1,
				},
				Expired: true,
			},
		},
		"error in page (embedded list options)": {
			url:          "/foo?page=-1",
			errorMessage: "negative page value",
		},
		"error in expired": {
			url:          "/foo?expired=foo",
			errorMessage: "Invalid expired",
		},
	}

	for name, tt := range carveListOptionsTests {
		t.Run(
			name, func(t *testing.T) {
				urlStruct, _ := url.Parse(tt.url)
				req := &http.Request{URL: urlStruct}
				opt, err := carveListOptionsFromRequest(req)

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
				assert.Equal(t, tt.carveListOptions, opt)
			},
		)
	}
}

func TestUserListOptionsFromRequest(t *testing.T) {
	userListOptionsTests := map[string]struct {
		// url string to parse
		url string
		// expected options
		userListOptions fleet.UserListOptions
		// expected error message, if any
		errorMessage string
	}{
		"no params passed": {
			url:             "/foo",
			userListOptions: fleet.UserListOptions{},
		},
		"embedded list options params defined": {
			url: "/foo?order_key=foo&order_direction=desc&page=1&per_page=100",
			userListOptions: fleet.UserListOptions{
				ListOptions: fleet.ListOptions{
					OrderKey:       "foo",
					OrderDirection: fleet.OrderDescending,
					Page:           1,
					PerPage:        100,
				},
			},
		},
		"all params defined": {
			url: "/foo?order_key=foo&order_direction=asc&page=10&per_page=1&team_id=1",
			userListOptions: fleet.UserListOptions{
				ListOptions: fleet.ListOptions{
					OrderKey:       "foo",
					OrderDirection: fleet.OrderAscending,
					Page:           10,
					PerPage:        1,
				},
				TeamID: 1,
			},
		},
		"error in page (embedded list options)": {
			url:          "/foo?page=-1",
			errorMessage: "negative page value",
		},
		"error in team_id (negative_number)": {
			url:          "/foo?team_id=-1",
			errorMessage: "Invalid team_id",
		},
		"error in team_id (not a number)": {
			url:          "/foo?team_id=foo",
			errorMessage: "Invalid team_id",
		},
	}

	for name, tt := range userListOptionsTests {
		t.Run(
			name, func(t *testing.T) {
				urlStruct, _ := url.Parse(tt.url)
				req := &http.Request{URL: urlStruct}
				opt, err := userListOptionsFromRequest(req)

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
				assert.Equal(t, tt.userListOptions, opt)
			},
		)
	}
}
