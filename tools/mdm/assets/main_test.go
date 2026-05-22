package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/testutils"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/spf13/cobra"
)

func newConfigManagerForTest() (*cobra.Command, config.Manager) {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().StringP("config", "c", "", "Path to a Fleet configuration file")
	return cmd, config.NewManager(cmd)
}

func TestFleetMysqlConfig(t *testing.T) {
	testutils.SaveEnv(t)
	os.Clearenv()

	configFile := filepath.Join(t.TempDir(), "fleet.yml")
	if err := os.WriteFile(configFile, []byte(`mysql:
  username: file_user
  password: file_pass
  address: file_addr:3306
  database: file_db
  tls_ca: /path/to/ca.pem
  tls_cert: /path/to/client-cert.pem
  tls_key: /path/to/client-key.pem
  tls_server_name: mysql.example.com
server:
  private_key: file_private_key
`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("loads MySQL config from Fleet config file", func(t *testing.T) {
		cmd, manager := newConfigManagerForTest()
		if err := cmd.ParseFlags([]string{"--config", configFile}); err != nil {
			t.Fatal(err)
		}

		got := manager.LoadConfig()
		if got.Mysql.Username != "file_user" {
			t.Fatalf("mysql username: got %q, want %q", got.Mysql.Username, "file_user")
		}
		if got.Mysql.Password != "file_pass" {
			t.Fatalf("mysql password: got %q, want %q", got.Mysql.Password, "file_pass")
		}
		if got.Mysql.Address != "file_addr:3306" {
			t.Fatalf("mysql address: got %q, want %q", got.Mysql.Address, "file_addr:3306")
		}
		if got.Mysql.Database != "file_db" {
			t.Fatalf("mysql database: got %q, want %q", got.Mysql.Database, "file_db")
		}
		if got.Mysql.TLSCA != "/path/to/ca.pem" {
			t.Fatalf("mysql tls ca: got %q, want %q", got.Mysql.TLSCA, "/path/to/ca.pem")
		}
		if got.Mysql.TLSCert != "/path/to/client-cert.pem" {
			t.Fatalf("mysql tls cert: got %q, want %q", got.Mysql.TLSCert, "/path/to/client-cert.pem")
		}
		if got.Mysql.TLSKey != "/path/to/client-key.pem" {
			t.Fatalf("mysql tls key: got %q, want %q", got.Mysql.TLSKey, "/path/to/client-key.pem")
		}
		if got.Mysql.TLSServerName != "mysql.example.com" {
			t.Fatalf("mysql tls server name: got %q, want %q", got.Mysql.TLSServerName, "mysql.example.com")
		}
	})

	t.Run("Fleet env vars override config file", func(t *testing.T) {
		t.Setenv("FLEET_MYSQL_USERNAME", "env_user")
		t.Setenv("FLEET_MYSQL_PASSWORD", "env_pass")

		cmd, manager := newConfigManagerForTest()
		if err := cmd.ParseFlags([]string{"--config", configFile}); err != nil {
			t.Fatal(err)
		}

		got := manager.LoadConfig()
		if got.Mysql.Username != "env_user" {
			t.Fatalf("mysql username: got %q, want %q", got.Mysql.Username, "env_user")
		}
		if got.Mysql.Password != "env_pass" {
			t.Fatalf("mysql password: got %q, want %q", got.Mysql.Password, "env_pass")
		}
		if got.Mysql.Address != "file_addr:3306" {
			t.Fatalf("mysql address: got %q, want %q", got.Mysql.Address, "file_addr:3306")
		}
	})

	t.Run("Fleet flags override env and config file", func(t *testing.T) {
		t.Setenv("FLEET_MYSQL_USERNAME", "env_user")

		cmd, manager := newConfigManagerForTest()
		if err := cmd.ParseFlags([]string{
			"--config", configFile,
			"--mysql_username", "flag_user",
			"--mysql_tls_config", "skip-verify",
		}); err != nil {
			t.Fatal(err)
		}

		got := manager.LoadConfig()
		if got.Mysql.Username != "flag_user" {
			t.Fatalf("mysql username: got %q, want %q", got.Mysql.Username, "flag_user")
		}
		if got.Mysql.TLSConfig != "skip-verify" {
			t.Fatalf("mysql tls config: got %q, want %q", got.Mysql.TLSConfig, "skip-verify")
		}
	})
}

func TestPrivateKeyFromOptions(t *testing.T) {
	t.Run("uses Fleet config when key flag is absent", func(t *testing.T) {
		got := privateKeyFromOptions(commandOptions{}, config.FleetConfig{
			Server: config.ServerConfig{PrivateKey: "config_private_key"},
		})
		if got != "config_private_key" {
			t.Fatalf("private key: got %q, want %q", got, "config_private_key")
		}
	})

	t.Run("key flag overrides Fleet config", func(t *testing.T) {
		got := privateKeyFromOptions(commandOptions{key: "flag_private_key"}, config.FleetConfig{
			Server: config.ServerConfig{PrivateKey: "config_private_key"},
		})
		if got != "flag_private_key" {
			t.Fatalf("private key: got %q, want %q", got, "flag_private_key")
		}
	})

	t.Run("truncates long keys to AES-256 length", func(t *testing.T) {
		got := privateKeyFromOptions(commandOptions{key: "123456789012345678901234567890123"}, config.FleetConfig{})
		if got != "12345678901234567890123456789012" {
			t.Fatalf("private key: got %q, want %q", got, "12345678901234567890123456789012")
		}
	})
}

func TestValidatePrivateKey(t *testing.T) {
	t.Run("requires key", func(t *testing.T) {
		if err := validatePrivateKey(""); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("requires AES-256 length", func(t *testing.T) {
		if err := validatePrivateKey("too-short"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("accepts 32 byte key", func(t *testing.T) {
		if err := validatePrivateKey("12345678901234567890123456789012"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNormalizeLegacyArgs(t *testing.T) {
	got := normalizeLegacyArgs([]string{
		"export",
		"-key=secret",
		"-dir", "mdm_assets",
		"-name", "vpp_token",
		"-db-user=fleet_user",
		"-db-password", "fleet_pass",
		"-db-address=localhost:3307",
		"-db-name", "fleet_db",
		"-tls-config=skip-verify",
		"-tls-ca", "/path/to/ca.pem",
		"-tls-cert=/path/to/cert.pem",
		"-tls-key", "/path/to/key.pem",
		"-tls-server-name=mysql.example.com",
	})
	want := []string{
		"export",
		"--key=secret",
		"--dir", "mdm_assets",
		"--name", "vpp_token",
		"--mysql_username=fleet_user",
		"--mysql_password", "fleet_pass",
		"--mysql_address=localhost:3307",
		"--mysql_database", "fleet_db",
		"--mysql_tls_config=skip-verify",
		"--mysql_tls_ca", "/path/to/ca.pem",
		"--mysql_tls_cert=/path/to/cert.pem",
		"--mysql_tls_key", "/path/to/key.pem",
		"--mysql_tls_server_name=mysql.example.com",
	}
	if len(got) != len(want) {
		t.Fatalf("normalized args length: got %d, want %d\n got: %#v\nwant: %#v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalized arg %d: got %q, want %q\n got: %#v\nwant: %#v", i, got[i], want[i], got, want)
		}
	}
}
