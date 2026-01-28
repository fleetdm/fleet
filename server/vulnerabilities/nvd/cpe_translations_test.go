package nvd

import (
	"slices"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func TestTranslate(t *testing.T) {
	tests := []struct {
		name         string
		translations CPETranslations
		software     fleet.Software
		matched      bool
		want         CPETranslation
	}{
		{
			name: "no match",
			translations: CPETranslations{
				{
					Software: CPETranslationSoftware{
						Name:   []string{"MyApp"},
						Source: []string{"apps"},
					},
					Filter: CPETranslation{
						Vendor: []string{"override"},
					},
				},
			},
			software: fleet.Software{
				Name:   "NotMyApp",
				Source: "apps",
			},
			matched: false,
			want:    CPETranslation{},
		},
		{
			name: "match on name and source",
			translations: CPETranslations{
				{
					Software: CPETranslationSoftware{
						Name:   []string{"MyApp"},
						Source: []string{"apps"},
					},
					Filter: CPETranslation{
						Vendor: []string{"mycompany"},
					},
				},
			},
			software: fleet.Software{
				Name:   "MyApp",
				Source: "apps",
			},
			matched: true,
			want: CPETranslation{
				Product: []string{"MyApp"},
				Vendor:  []string{"mycompany"},
			},
		},
		{
			name: "match on bundle identifier",
			translations: CPETranslations{
				{
					Software: CPETranslationSoftware{
						BundleIdentifier: []string{"com.mycompany.myapp"},
						Source:           []string{"apps"},
					},
					Filter: CPETranslation{
						Vendor:  []string{"mycompany"},
						Product: []string{"myapp"},
					},
				},
			},
			software: fleet.Software{
				Name:             "MyApp",
				BundleIdentifier: "com.mycompany.myapp",
				Source:           "apps",
			},
			matched: true,
			want: CPETranslation{
				Vendor:  []string{"mycompany"},
				Product: []string{"myapp"},
			},
		},
		{
			name: "match with regex",
			translations: CPETranslations{
				{
					Software: CPETranslationSoftware{
						Name:   []string{"/^My.*/"},
						Source: []string{"apps"},
					},
					Filter: CPETranslation{
						Vendor:  []string{"mycompany"},
						Product: []string{"myapp"},
					},
				},
			},
			software: fleet.Software{
				Name:   "MyApp",
				Source: "apps",
			},
			matched: true,
			want: CPETranslation{
				Vendor:  []string{"mycompany"},
				Product: []string{"myapp"},
			},
		},
		{
			name: "match with regex not matching",
			translations: CPETranslations{
				{
					Software: CPETranslationSoftware{
						Name:   []string{"/^My.*/"},
						Source: []string{"apps"},
					},
					Filter: CPETranslation{
						Vendor: []string{"mycompany"},
					},
				},
			},
			software: fleet.Software{
				Name:   "NotMyApp",
				Source: "apps",
			},
			matched: false,
			want:    CPETranslation{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reCache := newRegexpCache()
			got, matched, err := tt.translations.Translate(reCache, &tt.software)
			if err != nil {
				t.Fatalf("Translate() error = %v", err)
			}
			if matched != tt.matched {
				t.Errorf("Translate() matched = %v, want %v", matched, tt.matched)
			}
			if matched {
				if got.Part != tt.want.Part {
					t.Errorf("Translate() Part = %v, want %v", got.Part, tt.want.Part)
				}
				if !slices.Equal(got.Vendor, tt.want.Vendor) {
					t.Errorf("Translate() Vendor = %v, want %v", got.Vendor, tt.want.Vendor)
				}
				if !slices.Equal(got.Product, tt.want.Product) {
					t.Errorf("Translate() Product = %v, want %v", got.Product, tt.want.Product)
				}
				if !slices.Equal(got.TargetSW, tt.want.TargetSW) {
					t.Errorf("Translate() TargetSW = %v, want %v", got.TargetSW, tt.want.TargetSW)
				}
				if got.Skip != tt.want.Skip {
					t.Errorf("Translate() Skip = %v, want %v", got.Skip, tt.want.Skip)
				}
			}
		})
	}
}
