package appstate

import "github.com/fleetdm/fleet/server/kolide"

// Labels is the set of builtin labels that should be populated in the
// datastore
func Labels1() []kolide.Label {
	return []kolide.Label{
		{
			Name:      "All Hosts",
			Query:     "select 1;",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			Platform:  "darwin",
			Name:      "Mac OS X",
			Query:     "select 1 from osquery_info where build_platform = 'darwin';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			Platform:  "ubuntu",
			Name:      "Ubuntu Linux",
			Query:     "select 1 from osquery_info where build_platform = 'ubuntu';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			Platform:  "centos",
			Name:      "CentOS Linux",
			Query:     "select 1 from osquery_info where build_platform = 'centos';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			Platform:  "windows",
			Name:      "MS Windows",
			Query:     "select 1 from osquery_info where build_platform = 'windows';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
	}
}

func Labels2() []kolide.Label {
	return []kolide.Label{
		{
			Name:        "All Hosts",
			Query:       "select 1;",
			Description: "All hosts which have enrolled in Fleet",
			LabelType:   kolide.LabelTypeBuiltIn,
		},
		{
			Name:        "macOS",
			Query:       "select 1 from os_version where platform = 'darwin';",
			Description: "All macOS hosts",
			LabelType:   kolide.LabelTypeBuiltIn,
		},
		{
			Name:        "Ubuntu Linux",
			Query:       "select 1 from os_version where platform = 'ubuntu';",
			Description: "All Ubuntu hosts",
			LabelType:   kolide.LabelTypeBuiltIn,
		},
		{
			Name:        "CentOS Linux",
			Query:       "select 1 from os_version where platform = 'centos';",
			Description: "All CentOS hosts",
			LabelType:   kolide.LabelTypeBuiltIn,
		},
		{
			Name:        "MS Windows",
			Query:       "select 1 from os_version where platform = 'windows';",
			Description: "All Windows hosts",
			LabelType:   kolide.LabelTypeBuiltIn,
		},
	}
}
