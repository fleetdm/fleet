package fedora

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	models "github.com/vulsio/goval-dictionary/models/fedora"
)

func TestRpmNewPackageFromRpm(t *testing.T) {
	tests := []struct {
		name    string
		rpm     Rpm
		want    models.Package
		wantErr bool
	}{
		{
			name: "normal",
			rpm:  "name-1:1.0-1.module_12345.aarch64",
			want: models.Package{
				Name:     "name",
				Epoch:    "1",
				Version:  "1.0",
				Release:  "1.module_12345",
				Arch:     "aarch64",
				Filename: "name-1:1.0-1.module_12345.aarch64",
			},
		},
		{
			name: "with.rpm",
			rpm:  "name-1:1.0-1.module_12345.aarch64.rpm",
			want: models.Package{
				Name:     "name",
				Epoch:    "1",
				Version:  "1.0",
				Release:  "1.module_12345",
				Arch:     "aarch64",
				Filename: "name-1:1.0-1.module_12345.aarch64",
			},
		},
		{
			name: "name-with-hyphen",
			rpm:  "name-with-hyphen-1:1.0-1.module_12345.aarch64",
			want: models.Package{
				Name:     "name-with-hyphen",
				Epoch:    "1",
				Version:  "1.0",
				Release:  "1.module_12345",
				Arch:     "aarch64",
				Filename: "name-with-hyphen-1:1.0-1.module_12345.aarch64",
			},
		},
		{
			name:    "invalid rpm",
			rpm:     "invalid rpm",
			wantErr: true,
		},
		{
			name:    "can not find release",
			rpm:     "no_release:1.0.aarch64",
			wantErr: true,
		},
		{
			name:    "can not find version",
			rpm:     "no_version:1.0-1.module_12345.aarch64",
			wantErr: true,
		},
		{
			name: "can not find epoch",
			rpm:  "noepoch-1.0-1.module_12345.aarch64",
			want: models.Package{
				Name:     "noepoch",
				Epoch:    "0",
				Version:  "1.0",
				Release:  "1.module_12345",
				Arch:     "aarch64",
				Filename: "noepoch-1.0-1.module_12345.aarch64",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rpm.NewPackageFromRpm()
			if (err == nil) == tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("(-got +want):\n%s", diff)
			}
		})
	}
}

func TestUpdatesPerVersionMerge(t *testing.T) {
	tests := []struct {
		name   string
		source map[string]*models.Updates
		target map[string]*models.Updates
		want   map[string]*models.Updates
	}{
		{
			name: "merge success",
			source: map[string]*models.Updates{
				"35": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update35-1",
						},
						{
							Title: "update35-2",
						},
					},
				},
				"34": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update34-1",
						},
					},
				},
			},
			target: map[string]*models.Updates{
				"35": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update35-module-1",
						},
					},
				},
				"34": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update34-module-1",
						},
					},
				},
			},
			want: map[string]*models.Updates{
				"35": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update35-1",
						},
						{
							Title: "update35-2",
						},
						{
							Title: "update35-module-1",
						},
					},
				},
				"34": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update34-1",
						},
						{
							Title: "update34-module-1",
						},
					},
				},
			},
		},
		{
			name: "no panic when some version is missing",
			source: map[string]*models.Updates{
				"35": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update35-1",
						},
						{
							Title: "update35-2",
						},
					},
				},
				"34": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update34-1",
						},
					},
				},
			},
			target: map[string]*models.Updates{
				"35": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update35-module-1",
						},
					},
				},
			},
			want: map[string]*models.Updates{
				"35": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update35-1",
						},
						{
							Title: "update35-2",
						},
						{
							Title: "update35-module-1",
						},
					},
				},
				"34": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update34-1",
						},
					},
				},
			},
		},
		{
			name: "target is nil",
			source: map[string]*models.Updates{
				"39": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update39",
						},
					},
				},
			},
			target: nil,
			want: map[string]*models.Updates{
				"39": {
					UpdateList: []models.UpdateInfo{
						{
							Title: "update39",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeUpdates(tt.source, tt.target)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("(-got +want):\n%s", diff)
			}
		})
	}
}

func TestUniquePackages(t *testing.T) {
	opt := cmpopts.SortSlices((func(x, y models.Package) bool { return strings.Compare(x.Filename, y.Filename) > 0 }))
	tests := []struct {
		name string
		in   []models.Package
		want []models.Package
	}{
		{
			name: "normal",
			in: []models.Package{
				{Filename: "package1"},
				{Filename: "package2"},
				{Filename: "package2"},
				{Filename: "package3"},
				{Filename: "package3"},
				{Filename: "package3"},
			},
			want: []models.Package{
				{Filename: "package1"},
				{Filename: "package2"},
				{Filename: "package3"},
			},
		},
		{
			name: "no panic when it is blank",
			in:   []models.Package{},
			want: []models.Package{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniquePackages(tt.in)
			if diff := cmp.Diff(got, tt.want, opt); diff != "" {
				t.Errorf("(-got +want):\n%s", diff)
			}
		})
	}
}
