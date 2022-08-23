package msrc_input

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVulnXML(t *testing.T) {
	t.Run("VulnerabilityXML", func(t *testing.T) {
		t.Run("#PublishedDateEpoch", func(t *testing.T) {
			sut := VulnerabilityXML{
				Revisions: []RevisionHistoryXML{
					{
						Description: "<p>Information published.</p> ",
						Date:        "2022-05-10T07:00:00",
					},
				},
			}

			resultEpoch := sut.PublishedDateEpoch()
			require.NotNil(t, resultEpoch)

			resultDate := time.Unix(*resultEpoch, 0)
			require.Equal(t, 2022, resultDate.Year())
			require.Equal(t, time.May, resultDate.Month())
			require.Equal(t, 10, resultDate.Day())
		})

		t.Run("#IncludesVendorFix", func(t *testing.T) {
			t.Run("no remediations", func(t *testing.T) {
				sut := VulnerabilityXML{}
				require.False(t, sut.IncludesVendorFix("1"))
			})

			t.Run("no vendor fixes", func(t *testing.T) {
				sut := VulnerabilityXML{
					Remediations: []VulnerabilityRemediationXML{
						{
							Type:        "Known Issue",
							ProductIDs:  []string{"11896", "11897"},
							Description: "5013942",
						},
					},
				}

				require.False(t, sut.IncludesVendorFix("11896"))
			})

			t.Run("no vendor fix matches", func(t *testing.T) {
				sut := VulnerabilityXML{
					Remediations: []VulnerabilityRemediationXML{
						{
							Type:            "Vendor Fix",
							FixedBuild:      "10.0.17763.2928",
							ProductIDs:      []string{"11568", "11569"},
							Description:     "5013941",
							Supercedence:    "5012647",
							RestartRequired: "Yes",
						},
					},
				}

				require.False(t, sut.IncludesVendorFix("123"))
			})

			t.Run("vendor fix matches", func(t *testing.T) {
				sut := VulnerabilityXML{
					Remediations: []VulnerabilityRemediationXML{
						{
							Type:            "Vendor Fix",
							FixedBuild:      "10.0.17763.2928",
							ProductIDs:      []string{"11568", "11569"},
							Description:     "5013941",
							Supercedence:    "5012647",
							RestartRequired: "Yes",
						},
					},
				}

				require.True(t, sut.IncludesVendorFix("11568"))
			})
		})
	})
}
