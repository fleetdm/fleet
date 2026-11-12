package suse

import (
	"reflect"
	"testing"

	"github.com/k0kubun/pp"

	"github.com/vulsio/goval-dictionary/models"
)

func TestWalkSUSE(t *testing.T) {
	var tests = []struct {
		xmlName  string
		cri      Criteria
		tests    map[string]rpmInfoTest
		expected []distroPackage
	}{
		{
			cri: Criteria{
				Operator: "AND",
				Criterions: []Criterion{
					{
						Comment: "suse102 is installed",
					},
					{
						TestRef: "oval:org.opensuse.security:tst:99999999999",
						Comment: "mysql less than 5.0.26-16",
					},
				},
			},
			tests: map[string]rpmInfoTest{
				"oval:org.opensuse.security:tst:99999999999": {
					Name:         "mysql",
					FixedVersion: "0:5.0.26-16",
				},
			},
			expected: []distroPackage{
				{
					osVer: "10.2",
					pack: models.Package{
						Name:    "mysql",
						Version: "0:5.0.26-16",
					},
				},
			},
		},
		{
			cri: Criteria{
				Operator: "AND",
				Criterions: []Criterion{
					{
						Comment: "suse102 is installed",
					},
				},
				Criterias: []Criteria{
					{
						Operator: "OR",
						Criterions: []Criterion{
							{
								TestRef: "oval:org.opensuse.security:tst:99999999999",
								Comment: "mysql less than 5.0.26-16",
							},
						},
					},
				},
			},
			tests: map[string]rpmInfoTest{
				"oval:org.opensuse.security:tst:99999999999": {
					Name:         "mysql",
					FixedVersion: "0:5.0.26-16",
				},
			},
			expected: []distroPackage{
				{
					osVer: "10.2",
					pack: models.Package{
						Name:    "mysql",
						Version: "0:5.0.26-16",
					},
				},
			},
		},
		{
			xmlName: "opensuse.12.1.xml",
			cri: Criteria{
				Operator: "OR",
				Criterions: []Criterion{
					{
						TestRef: "oval:org.opensuse.security:tst:99999999999",
						Comment: "flash-player-11.2.202.243-30.1 is installed",
					},
				},
			},
			tests: map[string]rpmInfoTest{
				"oval:org.opensuse.security:tst:99999999999": {
					Name:         "flash-player",
					FixedVersion: "0:11.2.202.243-30.1",
				},
			},
			expected: []distroPackage{
				{
					osVer: "12.1",
					pack: models.Package{
						Name:    "flash-player",
						Version: "0:11.2.202.243-30.1",
					},
				},
			},
		},
		{
			cri: Criteria{
				Operator: "AND",
				Criterions: []Criterion{
					{
						Comment: "core9 is installed",
					},
					{
						TestRef: "oval:org.opensuse.security:tst:99999999999",
						Comment: "tar less than 1.13.25-325.10",
					},
				},
			},
			tests: map[string]rpmInfoTest{
				"oval:org.opensuse.security:tst:99999999999": {
					Name:         "tar",
					FixedVersion: "0:1.13.25-325.10",
				},
			},
			expected: []distroPackage{
				{
					osVer: "9",
					pack: models.Package{
						Name:    "tar",
						Version: "0:1.13.25-325.10",
					},
				},
			},
		},
		{
			cri: Criteria{
				Operator: "OR",
				Criterias: []Criteria{
					{
						Operator: "AND",
						Criterions: []Criterion{
							{
								Comment: "sles10-sp1 is installed",
							},
							{
								TestRef: "oval:org.opensuse.security:tst:99999999999",
								Comment: "openssl less than 0.9.8a-18.40.1",
							},
						},
					},
					{
						Operator: "AND",
						Criterions: []Criterion{
							{
								Comment: "sles10 is installed",
							},
						},
						Criterias: []Criteria{
							{
								Operator: "OR",
								Criterions: []Criterion{
									{
										TestRef: "oval:org.opensuse.security:tst:99999999998",
										Comment: "openssl less than 0.9.8a-18.39.3",
									},
								},
							},
						},
					},
				},
			},
			tests: map[string]rpmInfoTest{
				"oval:org.opensuse.security:tst:99999999999": {
					Name:         "openssl",
					FixedVersion: "0:0.9.8a-18.40.1",
				},
				"oval:org.opensuse.security:tst:99999999998": {
					Name:         "openssl",
					FixedVersion: "0:0.9.8a-18.39.3",
				},
			},
			expected: []distroPackage{
				{
					osVer: "10.1",
					pack: models.Package{
						Name:    "openssl",
						Version: "0:0.9.8a-18.40.1",
					},
				},
				{
					osVer: "10",
					pack: models.Package{
						Name:    "openssl",
						Version: "0:0.9.8a-18.39.3",
					},
				},
			},
		},
		{
			cri: Criteria{
				Operator: "AND",
				Criterions: []Criterion{
					{
						Comment: "openSUSE Leap 42.1 is installed",
					},
				},
				Criterias: []Criteria{
					{
						Operator: "OR",
						Criterias: []Criteria{
							{
								Operator: "AND",
								Criterions: []Criterion{
									{
										TestRef: "oval:org.opensuse.security:tst:99999999999",
										Comment: "bsdtar-3.1.2-13.2 is installed",
									},
									{
										TestRef: "oval:org.opensuse.security:tst:99999999998",
										Comment: "bsdtar is signed with openSUSE key",
									},
								},
							},
							{
								Operator: "AND",
								Criterions: []Criterion{
									{
										TestRef: "oval:org.opensuse.security:tst:99999999997",
										Comment: "libarchive-3.1.2-13.2 is installed",
									},
									{
										TestRef: "oval:org.opensuse.security:tst:99999999996",
										Comment: "libarchive is signed with openSUSE key",
									},
								},
							},
						},
					},
				},
			},
			tests: map[string]rpmInfoTest{
				"oval:org.opensuse.security:tst:99999999999": {
					Name:         "bsdtar",
					FixedVersion: "0:3.1.2-13.2",
				},
				"oval:org.opensuse.security:tst:99999999998": {
					Name: "bsdtar",
					SignatureKeyID: SignatureKeyid{
						Text: "b88b2fd43dbdc284",
					},
				},
				"oval:org.opensuse.security:tst:99999999997": {
					Name:         "libarchive",
					FixedVersion: "0:3.1.2-13.2",
				},
				"oval:org.opensuse.security:tst:99999999996": {
					Name: "libarchive",
					SignatureKeyID: SignatureKeyid{
						Text: "b88b2fd43dbdc284",
					},
				},
			},
			expected: []distroPackage{
				{
					osVer: "42.1",
					pack: models.Package{
						Name:    "bsdtar",
						Version: "0:3.1.2-13.2",
					},
				},
				{
					osVer: "42.1",
					pack: models.Package{
						Name:    "libarchive",
						Version: "0:3.1.2-13.2",
					},
				},
			},
		},
		{
			cri: Criteria{
				Operator: "OR",
				Criterias: []Criteria{
					{
						Operator: "AND",
						Criterias: []Criteria{
							{
								Operator: "OR",
								Criterions: []Criterion{
									{
										Comment: "SUSE Linux Enterprise Server 12 is installed",
									},
								},
							},
							{
								Operator: "OR",
								Criterions: []Criterion{
									{
										TestRef: "oval:org.opensuse.security:tst:99999999999",
										Comment: "bind-9.9.9P1-63.12.1 is installed",
									},
								},
							},
						},
					},
				},
			},
			tests: map[string]rpmInfoTest{
				"oval:org.opensuse.security:tst:99999999999": {
					Name:         "bind",
					FixedVersion: "0:9.9.9P1-63.12.1",
				},
			},
			expected: []distroPackage{
				{
					osVer: "12",
					pack: models.Package{
						Name:    "bind",
						Version: "0:9.9.9P1-63.12.1",
					},
				},
			},
		},
		{
			cri: Criteria{
				Operator: "AND",
				Criterias: []Criteria{
					{
						Operator: "OR",
						Criterions: []Criterion{
							{
								Comment: "SUSE Manager Proxy 4.0 is installed",
							},
							{
								Comment: "SUSE Linux Enterprise Micro 5.1 is installed",
							},
							{
								Comment: "SUSE Linux Enterprise Storage 7 is installed",
							},
						},
					},
				},
				Criterions: []Criterion{
					{
						TestRef: "oval:org.opensuse.security:tst:99999999999",
						Comment: "mailx-12.5-1.87 is installed",
					},
				},
			},
			tests: map[string]rpmInfoTest{
				"oval:org.opensuse.security:tst:99999999999": {
					Name:         "mailx",
					FixedVersion: "0:12.5-1.87",
				},
			},
			expected: []distroPackage{},
		},
		{
			cri: Criteria{
				Operator: "AND",
				Criterions: []Criterion{
					{
						Comment: "SUSE Linux Enterprise Server 12 is installed",
					},
					{
						TestRef: "oval:org.opensuse.security:tst:99999999999",
						Comment: "kernel-default is not affected",
					},
				},
			},
			tests: map[string]rpmInfoTest{
				"oval:org.opensuse.security:tst:99999999999": {
					Name: "kernel-default",
				},
			},
			expected: []distroPackage{},
		},
	}

	for i, tt := range tests {
		actual := collectSUSEPacks(tt.xmlName, tt.cri, tt.tests)
		if !reflect.DeepEqual(tt.expected, actual) {
			e := pp.Sprintf("%v", tt.expected)
			a := pp.Sprintf("%v", actual)
			t.Errorf("[%d]: expected: %s\n, actual: %s\n", i, e, a)
		}
	}
}

func TestGetOSVersion(t *testing.T) {
	var tests = []struct {
		s        string
		expected string
	}{
		{
			s:        "suse102",
			expected: "10.2",
		},
		{
			s:        "suse111-debug",
			expected: "11.1",
		},
		{
			s:        "core9",
			expected: "9",
		},
		{
			s:        "sles10",
			expected: "10",
		},
		{
			s:        "sles10-sp1",
			expected: "10.1",
		},
		{
			s:        "sles10-sp1-ltss",
			expected: "10.1",
		},
		{
			s:        "sles10-slepos",
			expected: "10",
		},
		{
			s:        "sled10",
			expected: "10",
		},
		{
			s:        "sled10-sp1",
			expected: "10.1",
		},
		{
			s:        "sled10-sp1-ltss",
			expected: "10.1",
		},
		{
			s:        "sled10-slepos",
			expected: "10",
		},
		{
			s:        "openSUSE 13.2",
			expected: "13.2",
		},
		{
			s:        "openSUSE 13.2 NonFree",
			expected: "13.2",
		},
		{
			s:        "openSUSE Tumbleweed",
			expected: "tumbleweed",
		},
		{
			s:        "openSUSE Leap 42.2",
			expected: "42.2",
		},
		{
			s:        "openSUSE Leap 42.2 NonFree",
			expected: "42.2",
		},
		{
			s:        "SUSE Linux Enterprise Server 12",
			expected: "12",
		},
		{
			s:        "SUSE Linux Enterprise Server 12-LTSS",
			expected: "12",
		},
		{
			s:        "SUSE Linux Enterprise Server 11-SECURITY",
			expected: "11",
		},
		{
			s:        "SUSE Linux Enterprise Server 11-CLIENT-TOOLS",
			expected: "11",
		},
		{
			s:        "SUSE Linux Enterprise Server 12 SP1",
			expected: "12.1",
		},
		{
			s:        "SUSE Linux Enterprise Server 12 SP1-LTSS",
			expected: "12.1",
		},
		{
			s:        "SUSE Linux Enterprise Server 11 SP1-CLIENT-TOOLS",
			expected: "11.1",
		},
		{
			s:        "SUSE Linux Enterprise Server for SAP Applications 12",
			expected: "12",
		},
		{
			s:        "SUSE Linux Enterprise Server for SAP Applications 12-LTSS",
			expected: "12",
		},
		{
			s:        "SUSE Linux Enterprise Server for SAP Applications 11-SECURITY",
			expected: "11",
		},
		{
			s:        "SUSE Linux Enterprise Server for SAP Applications 11-CLIENT-TOOLS",
			expected: "11",
		},
		{
			s:        "SUSE Linux Enterprise Server for SAP Applications 12 SP1",
			expected: "12.1",
		},
		{
			s:        "SUSE Linux Enterprise Server for SAP Applications 12 SP1-LTSS",
			expected: "12.1",
		},
		{
			s:        "SUSE Linux Enterprise Server for SAP Applications 11 SP1-CLIENT-TOOLS",
			expected: "11.1",
		},
		{
			s:        "SUSE Linux Enterprise Server for Python 2 15 SP1",
			expected: "15.1",
		},
		{
			s:        "SUSE Manager Proxy 4.0",
			expected: "",
		},
		{
			s:        "SUSE Linux Enterprise Micro 5.1",
			expected: "",
		},
		{
			s:        "SUSE Linux Enterprise Storage 7",
			expected: "",
		},
	}

	for i, tt := range tests {
		actual, err := getOSVersion(tt.s)
		if err != nil {
			t.Errorf("[%d] getOSVersion err: %s", i, err)
		}
		if tt.expected != actual {
			e := pp.Sprintf("%v", tt.expected)
			a := pp.Sprintf("%v", actual)
			t.Errorf("[%d]: expected: %s, actual: %s\n", i, e, a)
		}
	}
}
