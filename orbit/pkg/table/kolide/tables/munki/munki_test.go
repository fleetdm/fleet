//go:build darwin
// +build darwin

package munki

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMunkiReport(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name       string
		reportPath string

		expectedReportError  bool
		expectedReportNil    bool
		expectedReportFields map[string]string

		expectedInstallsError bool
		expectedInstallsNil   bool
		expectedInstallsRows  []map[string]string // Don't have any examples yet
	}{
		{
			name:       "normal",
			reportPath: "testdata/ManagedInstallReport.plist",
			expectedReportFields: map[string]string{
				"console_user": "auser",
			},
		},
		{
			name:                "missing file",
			reportPath:          "testdata/no such file",
			expectedReportNil:   true,
			expectedInstallsNil: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := New()
			m.reportPath = tt.reportPath
			mockQC := tablehelpers.MockQueryContext(nil)

			t.Run("generateMunkiReport", func(t *testing.T) {
				results, err := m.generateMunkiReport(ctx, mockQC)

				if tt.expectedReportError {
					require.Error(t, err)
					return
				}

				require.NoError(t, err)

				if tt.expectedReportNil {
					assert.Nil(t, results)
					return
				}

				if tt.expectedReportFields != nil {
					for k, v := range tt.expectedReportFields {
						assert.Equal(t, results[0][k], v, "Expected field %s matches", k)
					}
				}
			})

			t.Run("generateMunkiInstalls", func(t *testing.T) {
				results, err := m.generateMunkiInstalls(ctx, mockQC)

				if tt.expectedInstallsError {
					require.Error(t, err)
					return
				}

				require.NoError(t, err)

				if tt.expectedInstallsNil {
					assert.Nil(t, results)
					return
				}
			})
		})
	}
}
