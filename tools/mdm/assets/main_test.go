package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/spf13/viper"
)

func TestConfigPrecedence(t *testing.T) {
	// Create a temp directory with a .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envFile, []byte("USER=envfile_user\nPASSWORD=envfile_pass\nADDRESS=envfile_addr:3306\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Save and restore the original working directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Clean up any existing env vars for this test
	cleanEnv := func() {
		os.Unsetenv("ASSETS_DB_USER")
		os.Unsetenv("ASSETS_DB_PASSWORD")
		os.Unsetenv("ASSETS_DB_ADDRESS")
	}

	t.Run(".env file values are loaded", func(t *testing.T) {
		cleanEnv()
		cfg := viper.New()
		cfg.SetConfigFile(".env")
		cfg.SetConfigType("env")
		cfg.SetDefault("USER", "default_user")
		cfg.SetDefault("PASSWORD", "default_pass")
		cfg.SetDefault("ADDRESS", "default_addr:3306")
		cfg.AutomaticEnv()
		cfg.SetEnvPrefix("ASSETS_DB")
		_ = cfg.ReadInConfig()

		if got := cfg.GetString("USER"); got != "envfile_user" {
			t.Errorf("USER from .env: got %q, want %q", got, "envfile_user")
		}
		if got := cfg.GetString("PASSWORD"); got != "envfile_pass" {
			t.Errorf("PASSWORD from .env: got %q, want %q", got, "envfile_pass")
		}
	})

	t.Run("shell env vars override .env file", func(t *testing.T) {
		cleanEnv()
		os.Setenv("ASSETS_DB_USER", "shell_env_user")
		os.Setenv("ASSETS_DB_PASSWORD", "shell_env_pass")

		cfg := viper.New()
		cfg.SetConfigFile(".env")
		cfg.SetConfigType("env")
		cfg.SetDefault("USER", "default_user")
		cfg.SetDefault("PASSWORD", "default_pass")
		cfg.AutomaticEnv()
		cfg.SetEnvPrefix("ASSETS_DB")
		_ = cfg.ReadInConfig()

		if got := cfg.GetString("USER"); got != "shell_env_user" {
			t.Errorf("USER from shell env: got %q, want %q", got, "shell_env_user")
		}
		if got := cfg.GetString("PASSWORD"); got != "shell_env_pass" {
			t.Errorf("PASSWORD from shell env: got %q, want %q", got, "shell_env_pass")
		}
	})

	t.Run("defaults are used when nothing else is set", func(t *testing.T) {
		cleanEnv()

		cfg := viper.New()
		cfg.SetConfigFile(".env")
		cfg.SetConfigType("env")
		cfg.SetDefault("USER", "default_user")
		cfg.SetDefault("PASSWORD", "default_pass")
		cfg.AutomaticEnv()
		cfg.SetEnvPrefix("ASSETS_DB")
		// Don't read config (no .env file)
		cfg.SetConfigFile("/nonexistent/.env")

		if got := cfg.GetString("USER"); got != "default_user" {
			t.Errorf("USER default: got %q, want %q", got, "default_user")
		}
		if got := cfg.GetString("PASSWORD"); got != "default_pass" {
			t.Errorf("PASSWORD default: got %q, want %q", got, "default_pass")
		}
	})

	t.Run(".env file values override defaults", func(t *testing.T) {
		cleanEnv()

		cfg := viper.New()
		cfg.SetConfigFile(".env")
		cfg.SetConfigType("env")
		cfg.SetDefault("USER", "default_user")
		cfg.SetDefault("ADDRESS", "default_addr:3306")
		cfg.AutomaticEnv()
		cfg.SetEnvPrefix("ASSETS_DB")
		_ = cfg.ReadInConfig()

		// USER is in .env, ADDRESS is in .env
		if got := cfg.GetString("USER"); got != "envfile_user" {
			t.Errorf("USER from .env over default: got %q, want %q", got, "envfile_user")
		}
		if got := cfg.GetString("ADDRESS"); got != "envfile_addr:3306" {
			t.Errorf("ADDRESS from .env over default: got %q, want %q", got, "envfile_addr:3306")
		}
	})
}

func TestBuildTLSConfig(t *testing.T) {
	tests := []struct {
		name     string
		flags    struct {
			tlsConfig     string
			tlsCA         string
			tlsCert       string
			tlsKey        string
			tlsServerName string
		}
		want config.MysqlConfig
	}{
		{
			name: "skip-verify TLS config",
			flags: struct {
				tlsConfig     string
				tlsCA         string
				tlsCert       string
				tlsKey        string
				tlsServerName string
			}{tlsConfig: "skip-verify"},
			want: config.MysqlConfig{
				TLSConfig: "skip-verify",
			},
		},
		{
			name: "custom TLS with CA and server name",
			flags: struct {
				tlsConfig     string
				tlsCA         string
				tlsCert       string
				tlsKey        string
				tlsServerName string
			}{
				tlsConfig:     "custom",
				tlsCA:         "/etc/mysql/ca.pem",
				tlsCert:       "/etc/mysql/client-cert.pem",
				tlsKey:        "/etc/mysql/client-key.pem",
				tlsServerName: "mysql.example.com",
			},
			want: config.MysqlConfig{
				TLSConfig:     "custom",
				TLSCA:         "/etc/mysql/ca.pem",
				TLSCert:       "/etc/mysql/client-cert.pem",
				TLSKey:        "/etc/mysql/client-key.pem",
				TLSServerName: "mysql.example.com",
			},
		},
		{
			name:     "all empty",
			flags:    struct { tlsConfig, tlsCA, tlsCert, tlsKey, tlsServerName string }{},
			want:     config.MysqlConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global flag variables
			flagTLSConfig = tt.flags.tlsConfig
			flagTLSCA = tt.flags.tlsCA
			flagTLSCert = tt.flags.tlsCert
			flagTLSKey = tt.flags.tlsKey
			flagTLSServerName = tt.flags.tlsServerName

			got := buildTLSConfig()

			if got.TLSConfig != tt.want.TLSConfig {
				t.Errorf("TLSConfig: got %q, want %q", got.TLSConfig, tt.want.TLSConfig)
			}
			if got.TLSCA != tt.want.TLSCA {
				t.Errorf("TLSCA: got %q, want %q", got.TLSCA, tt.want.TLSCA)
			}
			if got.TLSCert != tt.want.TLSCert {
				t.Errorf("TLSCert: got %q, want %q", got.TLSCert, tt.want.TLSCert)
			}
			if got.TLSKey != tt.want.TLSKey {
				t.Errorf("TLSKey: got %q, want %q", got.TLSKey, tt.want.TLSKey)
			}
			if got.TLSServerName != tt.want.TLSServerName {
				t.Errorf("TLSServerName: got %q, want %q", got.TLSServerName, tt.want.TLSServerName)
			}
		})
	}
}
