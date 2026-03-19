package db

import (
	"reflect"
	"testing"

	"github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/models"
)

func Test_formatFamilyAndOSVer(t *testing.T) {
	type args struct {
		family string
		osVer  string
	}
	tests := []struct {
		in       args
		expected args
		wantErr  string
	}{
		{
			in: args{
				family: config.Debian,
				osVer:  "11",
			},
			expected: args{
				family: config.Debian,
				osVer:  "11",
			},
		},
		{
			in: args{
				family: config.Debian,
				osVer:  "11.1",
			},
			expected: args{
				family: config.Debian,
				osVer:  "11",
			},
		},
		{
			in: args{
				family: config.Ubuntu,
				osVer:  "20.04",
			},
			expected: args{
				family: config.Ubuntu,
				osVer:  "20.04",
			},
		},
		{
			in: args{
				family: config.Ubuntu,
				osVer:  "20.04.3",
			},
			expected: args{
				family: config.Ubuntu,
				osVer:  "20.04",
			},
		},
		{
			in: args{
				family: config.Raspbian,
				osVer:  "10",
			},
			expected: args{
				family: config.Debian,
				osVer:  "10",
			},
		},
		{
			in: args{
				family: config.Raspbian,
				osVer:  "10.1",
			},
			expected: args{
				family: config.Debian,
				osVer:  "10",
			},
		},
		{
			in: args{
				family: config.RedHat,
				osVer:  "8",
			},
			expected: args{
				family: config.RedHat,
				osVer:  "8",
			},
		},
		{
			in: args{
				family: config.RedHat,
				osVer:  "8.4",
			},
			expected: args{
				family: config.RedHat,
				osVer:  "8",
			},
		},
		{
			in: args{
				family: config.CentOS,
				osVer:  "8",
			},
			expected: args{
				family: config.RedHat,
				osVer:  "8",
			},
		},
		{
			in: args{
				family: config.CentOS,
				osVer:  "8.4",
			},
			expected: args{
				family: config.RedHat,
				osVer:  "8",
			},
		},
		{
			in: args{
				family: config.Oracle,
				osVer:  "8",
			},
			expected: args{
				family: config.Oracle,
				osVer:  "8",
			},
		},
		{
			in: args{
				family: config.Oracle,
				osVer:  "8.4",
			},
			expected: args{
				family: config.Oracle,
				osVer:  "8",
			},
		},
		{
			in: args{
				family: config.Amazon,
				osVer:  "1",
			},
			expected: args{
				family: config.Amazon,
				osVer:  "1",
			},
		},
		{
			in: args{
				family: config.Amazon,
				osVer:  "2",
			},
			expected: args{
				family: config.Amazon,
				osVer:  "2",
			},
		},
		{
			in: args{
				family: config.Amazon,
				osVer:  "2022",
			},
			expected: args{
				family: config.Amazon,
				osVer:  "2022",
			},
		},
		{
			in: args{
				family: config.Amazon,
				osVer:  "2023",
			},
			expected: args{
				family: config.Amazon,
				osVer:  "2023",
			},
		},
		{
			in: args{
				family: config.Alpine,
				osVer:  "3.15",
			},
			expected: args{
				family: config.Alpine,
				osVer:  "3.15",
			},
		},
		{
			in: args{
				family: config.Alpine,
				osVer:  "3.14",
			},
			expected: args{
				family: config.Alpine,
				osVer:  "3.14",
			},
		},
		{
			in: args{
				family: config.Alpine,
				osVer:  "3.14.1",
			},
			expected: args{
				family: config.Alpine,
				osVer:  "3.14",
			},
		},
		{
			in: args{
				family: config.OpenSUSE,
				osVer:  "10.2",
			},
			expected: args{
				family: config.OpenSUSE,
				osVer:  "10.2",
			},
		},
		{
			in: args{
				family: config.OpenSUSE,
				osVer:  "tumbleweed",
			},
			expected: args{
				family: config.OpenSUSE,
				osVer:  "tumbleweed",
			},
		},
		{
			in: args{
				family: config.OpenSUSELeap,
				osVer:  "15.3",
			},
			expected: args{
				family: config.OpenSUSELeap,
				osVer:  "15.3",
			},
		},
		{
			in: args{
				family: config.SUSEEnterpriseServer,
				osVer:  "15",
			},
			expected: args{
				family: config.SUSEEnterpriseServer,
				osVer:  "15",
			},
		},
		{
			in: args{
				family: config.SUSEEnterpriseDesktop,
				osVer:  "15",
			},
			expected: args{
				family: config.SUSEEnterpriseDesktop,
				osVer:  "15",
			},
		},
		{
			in: args{
				family: config.Fedora,
				osVer:  "35",
			},
			expected: args{
				family: config.Fedora,
				osVer:  "35",
			},
		},
		{
			in: args{
				family: "unknown",
				osVer:  "unknown",
			},
			wantErr: "Failed to detect family. err: unknown os family(unknown)",
		},
	}
	for i, tt := range tests {
		family, osVer, err := formatFamilyAndOSVer(tt.in.family, tt.in.osVer)
		if tt.wantErr != "" {
			if err.Error() != tt.wantErr {
				t.Errorf("[%d] formatFamilyAndOSVer expected: %#v\n  actual: %#v\n", i, tt.wantErr, err)
			}
		}

		if family != tt.expected.family || osVer != tt.expected.osVer {
			t.Errorf("[%d] formatFamilyAndOSVer expected: %#v\n  actual: %#v\n", i, tt.expected, args{family: family, osVer: osVer})
		}
	}
}

func Test_filterByRedHatMajor(t *testing.T) {
	type args struct {
		packs    []models.Package
		majorVer string
	}
	tests := []struct {
		in       args
		expected []models.Package
	}{
		{
			in: args{
				packs: []models.Package{
					{
						Name:    "name-el7",
						Version: "0:0.0.1-0.0.1.el7",
					},
					{
						Name:    "name-el8",
						Version: "0:0.0.1-0.0.1.el8",
					},
					{
						Name:    "name-module+el7",
						Version: "0:0.1.1-1.module+el7.1.0+7785+0ea9f177",
					},
					{
						Name:    "name-module+el8",
						Version: "0:0.1.1-1.module+el8.1.0+7785+0ea9f177",
					},
				},
				majorVer: "8",
			},
			expected: []models.Package{
				{
					Name:    "name-el8",
					Version: "0:0.0.1-0.0.1.el8",
				},
				{
					Name:    "name-module+el8",
					Version: "0:0.1.1-1.module+el8.1.0+7785+0ea9f177",
				},
			},
		},
		{
			in: args{
				packs: []models.Package{
					{
						Name:    "name-el7",
						Version: "0:0.0.1-0.0.1.el7",
					},
					{
						Name:    "name-el8",
						Version: "0:0.0.1-0.0.1.el8",
					},
					{
						Name:    "name-module+el7",
						Version: "0:0.1.1-1.module+el7.1.0+7785+0ea9f177",
					},
					{
						Name:    "name-module+el8",
						Version: "0:0.1.1-1.module+el8.1.0+7785+0ea9f177",
					},
				},
				majorVer: "",
			},
			expected: []models.Package{
				{
					Name:    "name-el7",
					Version: "0:0.0.1-0.0.1.el7",
				},
				{
					Name:    "name-el8",
					Version: "0:0.0.1-0.0.1.el8",
				},
				{
					Name:    "name-module+el7",
					Version: "0:0.1.1-1.module+el7.1.0+7785+0ea9f177",
				},
				{
					Name:    "name-module+el8",
					Version: "0:0.1.1-1.module+el8.1.0+7785+0ea9f177",
				},
			},
		},
		{
			in: args{
				packs: []models.Package{
					{
						Name:    "name-el7",
						Version: "0:0.0.1-0.0.1.el7",
					},
					{
						Name:    "name-el8",
						Version: "0:0.0.1-0.0.1.el8",
					},
					{
						Name:        "notfixedyet",
						NotFixedYet: true,
					},
				},
				majorVer: "8",
			},
			expected: []models.Package{
				{
					Name:    "name-el8",
					Version: "0:0.0.1-0.0.1.el8",
				},
				{
					Name:        "notfixedyet",
					NotFixedYet: true,
				},
			},
		},
	}

	for i, tt := range tests {
		if aout := filterByRedHatMajor(tt.in.packs, tt.in.majorVer); !reflect.DeepEqual(aout, tt.expected) {
			t.Errorf("[%d] filterByRedHatMajor expected: %#v\n  actual: %#v\n", i, tt.expected, aout)
		}
	}
}
