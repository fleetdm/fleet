package service

import (
	"encoding/json"
	"testing"

	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyOptionsYaml(t *testing.T) {
	var testCases = []struct {
		yml       string
		options   *kolide.OptionsSpec
		shouldErr bool
	}{
		{"notyaml", nil, true},
		{
			yml: `
apiVersion: k8s.kolide.com/v1alpha1
kind: OsqueryQuery
spec:
  name: osquery_schedule
  description: Report performance stats
  query: select * from osquery_schedule
`, // Wrong kind of yaml
			options:   nil,
			shouldErr: true,
		},
		{
			yml: `
apiVersion: k8s.kolide.com/v1alpha1
kind: OsqueryOptions
spec:
  config:
    foo: bar
  overrides:
    # Note configs in overrides take precedence over base configs
    platforms:
      darwin:
        bing: bang
`,
			options: &kolide.OptionsSpec{
				Config: json.RawMessage(`{"foo":"bar"}`),
				Overrides: kolide.OptionsOverrides{
					Platforms: map[string]json.RawMessage{
						"darwin": json.RawMessage(`{"bing":"bang"}`),
					},
				},
			},
			shouldErr: false,
		},
	}

	var gotOptions *kolide.OptionsSpec
	ds := &mock.Store{
		OsqueryOptionsStore: mock.OsqueryOptionsStore{
			ApplyOptionsFunc: func(options *kolide.OptionsSpec) error {
				gotOptions = options
				return nil
			},
		},
	}
	svc := service{
		ds: ds,
	}

	for _, tt := range testCases {
		gotOptions = nil
		t.Run("", func(t *testing.T) {
			err := svc.ApplyOptionsYaml(tt.yml)
			if tt.shouldErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.options, gotOptions)
			}
		})
	}
}

func TestOptionsYamlRoundtrip(t *testing.T) {
	var testCases = []struct {
		spec kolide.OptionsSpec
	}{
		{
			kolide.OptionsSpec{
				json.RawMessage(`{"foo":"bar"}`),
				kolide.OptionsOverrides{},
			},
		},
		{
			kolide.OptionsSpec{
				json.RawMessage(`{"bing":"bang","boom":2}`),
				kolide.OptionsOverrides{
					map[string]json.RawMessage{
						"foobar":    json.RawMessage(`{"manzanita":"scratch"}`),
						"froobling": json.RawMessage(`{"doornail":"mumble"}`),
					},
				},
			},
		},
	}

	var returnOptions, gotOptions *kolide.OptionsSpec
	ds := &mock.Store{
		OsqueryOptionsStore: mock.OsqueryOptionsStore{
			GetOptionsFunc: func() (*kolide.OptionsSpec, error) {
				return returnOptions, nil
			},
			ApplyOptionsFunc: func(options *kolide.OptionsSpec) error {
				gotOptions = options
				return nil
			},
		},
	}
	svc := service{
		ds: ds,
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			returnOptions = &tt.spec
			gotOptions = nil

			yml, err := svc.GetOptionsYaml()
			require.Nil(t, err)

			err = svc.ApplyOptionsYaml(yml)
			require.Nil(t, err)

			assert.Equal(t, returnOptions, gotOptions)

		})
	}
}
