package ubuntu

import (
	"reflect"
	"testing"

	"github.com/k0kubun/pp"

	"github.com/vulsio/goval-dictionary/models"
)

func TestCollectUbuntuPacks(t *testing.T) {
	var tests = []struct {
		cri      Criteria
		tests    map[string]dpkgInfoTest
		expected []models.Package
	}{
		{
			cri: Criteria{
				Criterions: []Criterion{
					{
						TestRef: "oval:com.ubuntu.jammy:tst:200224390000040",
						Comment: "gcc-snapshot package in jammy, is related to the CVE in some way and has been fixed (note: '20140405-0ubuntu1').",
					},
				},
			},
			tests: map[string]dpkgInfoTest{
				"oval:com.ubuntu.jammy:tst:200224390000040": {
					Name:         "gcc-snapshot",
					FixedVersion: "0:20140405-0ubuntu1",
				},
			},
			expected: []models.Package{},
		},
		{
			cri: Criteria{
				Criterions: []Criterion{
					{
						TestRef: "oval:com.ubuntu.jammy:tst:2018128860000050",
						Comment: "gcc-snapshot: while related to the CVE in some way, a decision has been made to ignore this issue.",
					},
				},
			},
			tests: map[string]dpkgInfoTest{
				"oval:com.ubuntu.jammy:tst:2018128860000050": {Name: "gcc-snapshot"},
			},
			expected: []models.Package{
				{
					Name:        "gcc-snapshot",
					NotFixedYet: true,
				},
			},
		},
		{
			cri: Criteria{
				Criterions: []Criterion{
					{
						TestRef: "oval:com.ubuntu.jammy:tst:200224390000000",
						Comment: "gcc-arm-none-eabi package in jammy is affected and may need fixing.",
					},
				},
			},
			tests: map[string]dpkgInfoTest{
				"oval:com.ubuntu.jammy:tst:200224390000000": {Name: "gcc-arm-none-eabi"},
			},
			expected: []models.Package{},
		},
		{
			cri: Criteria{
				Criterions: []Criterion{
					{
						TestRef: "oval:com.ubuntu.jammy:tst:200224390000000",
						Comment: "gcc-arm-none-eabi package in jammy is affected and needs fixing.",
					},
				},
			},
			tests: map[string]dpkgInfoTest{
				"oval:com.ubuntu.jammy:tst:200224390000000": {Name: "gcc-arm-none-eabi"},
			},
			expected: []models.Package{
				{
					Name:        "gcc-arm-none-eabi",
					NotFixedYet: true,
				},
			},
		},
		{
			cri: Criteria{
				Criterions: []Criterion{
					{
						TestRef: "oval:com.ubuntu.jammy:tst:2018128860000050",
						Comment: "gcc-snapshot package in jammy is affected, but a decision has been made to defer addressing it.",
					},
				},
			},
			tests: map[string]dpkgInfoTest{
				"oval:com.ubuntu.jammy:tst:2018128860000050": {Name: "gcc-snapshot"},
			},
			expected: []models.Package{
				{
					Name:        "gcc-snapshot",
					NotFixedYet: true,
				},
			},
		},
		{
			cri: Criteria{
				Criterions: []Criterion{
					{
						TestRef: "oval:com.ubuntu.xenial:tst:2020143720000010",
						Comment: "grub2-unsigned package in xenial is affected. An update containing the fix has been completed and is pending publication (note: '2.04-1ubuntu42').",
					},
				},
			},
			tests: map[string]dpkgInfoTest{
				"oval:com.ubuntu.xenial:tst:2020143720000010": {
					Name:         "grub2-unsigned",
					FixedVersion: "0:2.04-1ubuntu42",
				},
			},
			expected: []models.Package{
				{
					Name:        "grub2-unsigned",
					NotFixedYet: true,
				},
			},
		},
		{
			cri: Criteria{
				Criterions: []Criterion{
					{
						TestRef: "oval:com.ubuntu.jammy:tst:2021285440000000",
						Comment: "subversion package in jammy was vulnerable but has been fixed (note: '1.14.1-3ubuntu0.22.04.1').",
					},
				},
			},
			tests: map[string]dpkgInfoTest{
				"oval:com.ubuntu.jammy:tst:2021285440000000": {
					Name:         "subversion",
					FixedVersion: "0:1.14.1-3ubuntu0.22.04.1",
				},
			},
			expected: []models.Package{
				{
					Name:        "subversion",
					Version:     "0:1.14.1-3ubuntu0.22.04.1",
					NotFixedYet: false,
				},
			},
		},
		{
			cri: Criteria{
				Criterions: []Criterion{
					{
						TestRef: "oval:com.ubuntu.jammy:tst:2021299550000000",
						Comment: "firefox package in jammy was vulnerable and has been fixed, but no release version available for it.",
					},
				},
			},
			tests: map[string]dpkgInfoTest{
				"oval:com.ubuntu.jammy:tst:2021299550000000": {Name: "firefox"},
			},
			expected: []models.Package{
				{
					Name:        "firefox",
					Version:     "",
					NotFixedYet: false,
				},
			},
		},
		{
			cri: Criteria{
				Criterions: []Criterion{
					{
						TestRef: "oval:com.ubuntu.jammy:tst:2016107230000000",
						Comment: "Is kernel linux running",
					},
					{
						TestRef: "oval:com.ubuntu.jammy:tst:2018121260000030",
						Comment: "kernel version comparison",
					},
				},
			},
			tests:    map[string]dpkgInfoTest{},
			expected: []models.Package{},
		},
		{
			cri: Criteria{
				Criterias: []Criteria{
					{
						Criterions: []Criterion{
							{
								TestRef: "oval:com.ubuntu.jammy:tst:200901660000010",
								Comment: "poppler package in jammy was vulnerable but has been fixed (note: '0.10.5-1ubuntu2').",
							},
						},
					},
				},
			},
			tests: map[string]dpkgInfoTest{
				"oval:com.ubuntu.jammy:tst:200901660000010": {
					Name:         "poppler",
					FixedVersion: "0:0.10.5-1ubuntu2",
				},
			},
			expected: []models.Package{
				{
					Name:        "poppler",
					Version:     "0:0.10.5-1ubuntu2",
					NotFixedYet: false,
				},
			},
		},
	}

	for i, tt := range tests {
		if actual := collectUbuntuPacks(tt.cri, tt.tests); !reflect.DeepEqual(tt.expected, actual) {
			e := pp.Sprintf("%v", tt.expected)
			a := pp.Sprintf("%v", actual)
			t.Errorf("[%d]: expected: %s\n, actual: %s\n", i, e, a)
		}
	}
}
