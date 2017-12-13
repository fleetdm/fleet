package kolide

import (
	"testing"

	"github.com/ghodss/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalYaml(t *testing.T) {
	y := []byte(`
apiVersion: k8s.kolide.com/v1alpha1
kind: OsqueryOptions
spec:
  config:
    options:
      distributed_interval: 3
      distributed_tls_max_attempts: 3
      logger_plugin: tls
      logger_tls_endpoint: /api/v1/osquery/log
      logger_tls_period: 10
  overrides:
    platforms:
      darwin:
        options:
          distributed_interval: 10
          distributed_tls_max_attempts: 10
          logger_plugin: tls
          logger_tls_endpoint: /api/v1/osquery/log
          logger_tls_period: 300
          disable_tables: chrome_extensions
          docker_socket: /var/run/docker.sock
        file_paths:
          users:
            - /Users/%/Library/%%
            - /Users/%/Documents/%%
          etc:
            - /etc/%%
      linux:
        options:
          distributed_interval: 10
          distributed_tls_max_attempts: 3
          logger_plugin: tls
          logger_tls_endpoint: /api/v1/osquery/log
          logger_tls_period: 60
          schedule_timeout: 60
          docker_socket: /etc/run/docker.sock
        frobulations:
          - fire
          - ice
`)

	expectedConfig := `{
  "options":{
    "distributed_interval":3,
    "distributed_tls_max_attempts":3,
    "logger_plugin":"tls",
    "logger_tls_endpoint":"/api/v1/osquery/log",
    "logger_tls_period":10
  }
}`

	expectedDarwin := `{
  "options":{
    "disable_tables":"chrome_extensions",
    "distributed_interval":10,
    "distributed_tls_max_attempts":10,
    "docker_socket":"/var/run/docker.sock",
    "logger_plugin":"tls",
    "logger_tls_endpoint":"/api/v1/osquery/log",
    "logger_tls_period":300
  },
  "file_paths":{
    "etc":[
      "/etc/%%"
    ],
    "users":[
      "/Users/%/Library/%%",
      "/Users/%/Documents/%%"
    ]
  }
}`

	expectedLinux := `{
  "options":{
    "distributed_interval":10,
    "distributed_tls_max_attempts":3,
    "docker_socket":"/etc/run/docker.sock",
    "logger_plugin":"tls",
    "logger_tls_endpoint":"/api/v1/osquery/log",
    "logger_tls_period":60,
    "schedule_timeout":60
  },
  "frobulations": [
    "fire",
    "ice"
  ]
}`

	var foo OptionsYaml
	err := yaml.Unmarshal(y, &foo)

	require.Nil(t, err)

	assert.JSONEq(t, expectedConfig, string(foo.Spec.Config))

	platformOverrides := foo.Spec.Overrides.Platforms
	assert.Len(t, platformOverrides, 2)

	if assert.Contains(t, platformOverrides, "darwin") {
		assert.JSONEq(t, expectedDarwin, string(platformOverrides["darwin"]))
	}

	if assert.Contains(t, platformOverrides, "linux") {
		assert.JSONEq(t, expectedLinux, string(platformOverrides["linux"]))
	}
}
