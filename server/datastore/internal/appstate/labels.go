package appstate

import "github.com/fleetdm/fleet/v4/server/fleet"

// Labels is the set of builtin labels that should be populated in the
// datastore
func Labels1() []fleet.Label {
	return []fleet.Label{
		{
			Name:      "All Hosts",
			Query:     "select 1;",
			LabelType: fleet.LabelTypeBuiltIn,
		},
		{
			Platform:  "darwin",
			Name:      "Mac OS X",
			Query:     "select 1 from osquery_info where build_platform = 'darwin';",
			LabelType: fleet.LabelTypeBuiltIn,
		},
		{
			Platform:  "ubuntu",
			Name:      "Ubuntu Linux",
			Query:     "select 1 from osquery_info where build_platform = 'ubuntu';",
			LabelType: fleet.LabelTypeBuiltIn,
		},
		{
			Platform:  "centos",
			Name:      "CentOS Linux",
			Query:     "select 1 from osquery_info where build_platform = 'centos';",
			LabelType: fleet.LabelTypeBuiltIn,
		},
		{
			Platform:  "windows",
			Name:      "MS Windows",
			Query:     "select 1 from osquery_info where build_platform = 'windows';",
			LabelType: fleet.LabelTypeBuiltIn,
		},
	}
}

func Labels2() []fleet.Label {
	return []fleet.Label{
		{
			Name:        "All Hosts",
			Query:       "select 1;",
			Description: "All hosts which have enrolled in Fleet",
			LabelType:   fleet.LabelTypeBuiltIn,
		},
		{
			Name:        "macOS",
			Query:       "select 1 from os_version where platform = 'darwin';",
			Description: "All macOS hosts",
			LabelType:   fleet.LabelTypeBuiltIn,
		},
		{
			Name:        "Ubuntu Linux",
			Query:       "select 1 from os_version where platform = 'ubuntu';",
			Description: "All Ubuntu hosts",
			LabelType:   fleet.LabelTypeBuiltIn,
		},
		{
			Name:        "CentOS Linux",
			Query:       "select 1 from os_version where platform = 'centos';",
			Description: "All CentOS hosts",
			LabelType:   fleet.LabelTypeBuiltIn,
		},
		{
			Name:        "MS Windows",
			Query:       "select 1 from os_version where platform = 'windows';",
			Description: "All Windows hosts",
			LabelType:   fleet.LabelTypeBuiltIn,
		},
	}
}
