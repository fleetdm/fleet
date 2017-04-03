package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	envPrefix = "KOLIDE"
)

// MysqlConfig defines configs related to MySQL
type MysqlConfig struct {
	Address       string
	Username      string
	Password      string
	Database      string
	TLSCert       string `yaml:"tls_cert"`
	TLSKey        string `yaml:"tls_key"`
	TLSCA         string `yaml:"tls_ca"`
	TLSServerName string `yaml:"tls_server_name"`
	TLSConfig     string `yaml:"tls_config"` //tls=customValue in DSN
}

// RedisConfig defines configs related to Redis
type RedisConfig struct {
	Address  string
	Password string
}

const (
	TLSProfileKey          = "server.tls_compatibility"
	TLSProfileModern       = "modern"
	TLSProfileIntermediate = "intermediate"
	TLSProfileOld          = "old"
)

// ServerConfig defines configs related to the Kolide server
type ServerConfig struct {
	Address    string
	Cert       string
	Key        string
	TLS        bool
	TLSProfile string
}

// AuthConfig defines configs related to user authorization
type AuthConfig struct {
	JwtKey      string `yaml:"jwt_key"`
	BcryptCost  int    `yaml:"bcrypt_cost"`
	SaltKeySize int    `yaml:"salt_key_size"`
}

// AppConfig defines configs related to HTTP
type AppConfig struct {
	TokenKeySize              int           `yaml:"token_key_size"`
	TokenKey                  string        `yaml:"token_key"`
	InviteTokenValidityPeriod time.Duration `yaml:"invite_token_validity_period"`
}

// SessionConfig defines configs related to user sessions
type SessionConfig struct {
	KeySize  int `yaml:"key_size"`
	Duration time.Duration
}

// OsqueryConfig defines configs related to osquery
type OsqueryConfig struct {
	NodeKeySize         int           `yaml:"node_key_size"`
	StatusLogFile       string        `yaml:"status_log_file"`
	ResultLogFile       string        `yaml:"result_log_file"`
	EnableLogRotation   bool          `yaml:"enable_log_rotation"`
	LabelUpdateInterval time.Duration `yaml:"label_update_interval"`
}

// LoggingConfig defines configs related to logging
type LoggingConfig struct {
	Debug         bool
	JSON          bool
	DisableBanner bool `yaml:"disable_banner"`
}

// KolideConfig stores the application configuration. Each subcategory is
// broken up into it's own struct, defined above. When editing any of these
// structs, Manager.addConfigs and Manager.LoadConfig should be
// updated to set and retrieve the configurations as appropriate.
type KolideConfig struct {
	Mysql   MysqlConfig
	Redis   RedisConfig
	Server  ServerConfig
	Auth    AuthConfig
	App     AppConfig
	Session SessionConfig
	Osquery OsqueryConfig
	Logging LoggingConfig
}

// addConfigs adds the configuration keys and default values that will be
// filled into the KolideConfig struct
func (man Manager) addConfigs() {
	// MySQL
	man.addConfigString("mysql.address", "localhost:3306",
		"MySQL server address (host:port)")
	man.addConfigString("mysql.username", "kolide",
		"MySQL server username")
	man.addConfigString("mysql.password", "kolide",
		"MySQL server password (prefer env variable for security)")
	man.addConfigString("mysql.database", "kolide",
		"MySQL database name")
	man.addConfigString("mysql.tls_cert", "",
		"MySQL TLS client certificate path")
	man.addConfigString("mysql.tls_key", "",
		"MySQL TLS client key path")
	man.addConfigString("mysql.tls_ca", "",
		"MySQL TLS server CA")
	man.addConfigString("mysql.tls_server_name", "",
		"MySQL TLS server name")
	man.addConfigString("mysql.tls_config", "",
		"MySQL TLS config value. Use skip-verify, true, false or custom key.")

	// Redis
	man.addConfigString("redis.address", "localhost:6379",
		"Redis server address (host:port)")
	man.addConfigString("redis.password", "",
		"Redis server password (prefer env variable for security)")

	// Server
	man.addConfigString("server.address", "0.0.0.0:8080",
		"Kolide server address (host:port)")
	man.addConfigString("server.cert", "./tools/osquery/kolide.crt",
		"Kolide TLS certificate path")
	man.addConfigString("server.key", "./tools/osquery/kolide.key",
		"Kolide TLS key path")
	man.addConfigBool("server.tls", true,
		"Enable TLS (required for osqueryd communication)")
	man.addConfigString(TLSProfileKey, TLSProfileModern,
		fmt.Sprintf("TLS security profile choose one of %s, %s or %s",
			TLSProfileModern, TLSProfileIntermediate, TLSProfileOld))

	// Auth
	man.addConfigString(
		"auth.jwt_key", "CHANGEME", "JWT session token key")
	man.addConfigInt("auth.bcrypt_cost", 12,
		"Bcrypt iterations")
	man.addConfigInt("auth.salt_key_size", 24,
		"Size of salt for passwords")

	// App
	man.addConfigString("app.token_key", "CHANGEME",
		"Secret key for generating invite and reset tokens")
	man.addConfigDuration("app.invite_token_validity_period", 5*24*time.Hour,
		"Duration invite tokens remain valid (i.e. 1h)")
	man.addConfigInt("app.token_key_size", 24,
		"Size of generated tokens")

	// Session
	man.addConfigInt("session.key_size", 64,
		"Size of generated session keys")
	man.addConfigDuration("session.duration", 24*90*time.Hour,
		"Duration session keys remain valid (i.e. 24h)")

	// Osquery
	man.addConfigInt("osquery.node_key_size", 24,
		"Size of generated osqueryd node keys")
	man.addConfigString("osquery.status_log_file", "/tmp/osquery_status",
		"Path for osqueryd status logs")
	man.addConfigString("osquery.result_log_file", "/tmp/osquery_result",
		"Path for osqueryd result logs")
	man.addConfigDuration("osquery.label_update_interval", 1*time.Hour,
		"Interval to update host label membership (i.e. 1h)")
	man.addConfigBool("osquery.enable_log_rotation", false,
		"Osquery log files will be automatically rotated")

	// Logging
	man.addConfigBool("logging.debug", false,
		"Enable debug logging")
	man.addConfigBool("logging.json", false,
		"Log in JSON format")
	man.addConfigBool("logging.disable_banner", false,
		"Disable startup banner")
}

// LoadConfig will load the config variables into a fully initialized
// KolideConfig struct
func (man Manager) LoadConfig() KolideConfig {
	man.loadConfigFile()

	return KolideConfig{
		Mysql: MysqlConfig{
			Address:       man.getConfigString("mysql.address"),
			Username:      man.getConfigString("mysql.username"),
			Password:      man.getConfigString("mysql.password"),
			Database:      man.getConfigString("mysql.database"),
			TLSCert:       man.getConfigString("mysql.tls_cert"),
			TLSKey:        man.getConfigString("mysql.tls_key"),
			TLSCA:         man.getConfigString("mysql.tls_ca"),
			TLSServerName: man.getConfigString("mysql.tls_server_name"),
			TLSConfig:     man.getConfigString("mysql.tls_config"),
		},
		Redis: RedisConfig{
			Address:  man.getConfigString("redis.address"),
			Password: man.getConfigString("redis.password"),
		},
		Server: ServerConfig{
			Address:    man.getConfigString("server.address"),
			Cert:       man.getConfigString("server.cert"),
			Key:        man.getConfigString("server.key"),
			TLS:        man.getConfigBool("server.tls"),
			TLSProfile: man.getConfigTLSProfile(),
		},
		Auth: AuthConfig{
			JwtKey:      man.getConfigString("auth.jwt_key"),
			BcryptCost:  man.getConfigInt("auth.bcrypt_cost"),
			SaltKeySize: man.getConfigInt("auth.salt_key_size"),
		},
		App: AppConfig{
			TokenKeySize:              man.getConfigInt("app.token_key_size"),
			TokenKey:                  man.getConfigString("app.token_key"),
			InviteTokenValidityPeriod: man.getConfigDuration("app.invite_token_validity_period"),
		},
		Session: SessionConfig{
			KeySize:  man.getConfigInt("session.key_size"),
			Duration: man.getConfigDuration("session.duration"),
		},
		Osquery: OsqueryConfig{
			NodeKeySize:         man.getConfigInt("osquery.node_key_size"),
			StatusLogFile:       man.getConfigString("osquery.status_log_file"),
			ResultLogFile:       man.getConfigString("osquery.result_log_file"),
			LabelUpdateInterval: man.getConfigDuration("osquery.label_update_interval"),
			EnableLogRotation:   man.getConfigBool("osquery.enable_log_rotation"),
		},
		Logging: LoggingConfig{
			Debug:         man.getConfigBool("logging.debug"),
			JSON:          man.getConfigBool("logging.json"),
			DisableBanner: man.getConfigBool("logging.disable_banner"),
		},
	}
}

// IsSet determines whether a given config key has been explicitly set by any
// of the configuration sources. If false, the default value is being used.
func (man Manager) IsSet(key string) bool {
	return man.viper.IsSet(key)
}

// envNameFromConfigKey converts a config key into the corresponding
// environment variable name
func envNameFromConfigKey(key string) string {
	return envPrefix + "_" + strings.ToUpper(strings.Replace(key, ".", "_", -1))
}

// flagNameFromConfigKey converts a config key into the corresponding flag name
func flagNameFromConfigKey(key string) string {
	return strings.Replace(key, ".", "_", -1)
}

// Manager manages the addition and retrieval of config values for Kolide
// configs. It's only public API method is LoadConfig, which will return the
// populated KolideConfig struct.
type Manager struct {
	viper    *viper.Viper
	command  *cobra.Command
	defaults map[string]interface{}
}

// NewManager initializes a Manager wrapping the provided cobra
// command. All config flags will be attached to that command (and inherited by
// the subcommands). Typically this should be called just once, with the root
// command.
func NewManager(command *cobra.Command) Manager {
	man := Manager{
		viper:    viper.New(),
		command:  command,
		defaults: map[string]interface{}{},
	}
	man.addConfigs()
	return man
}

// addDefault will check for duplication, then add a default value to the
// defaults map
func (man Manager) addDefault(key string, defVal interface{}) {
	if _, exists := man.defaults[key]; exists {
		panic("Trying to add duplicate config for key " + key)
	}

	man.defaults[key] = defVal
}

func getFlagUsage(key string, usage string) string {
	return fmt.Sprintf("Env: %s\n\t\t%s", envNameFromConfigKey(key), usage)
}

// getInterfaceVal is a helper function used by the getConfig* functions to
// retrieve the config value as interface{}, which will then be cast to the
// appropriate type by the getConfig* function.
func (man Manager) getInterfaceVal(key string) interface{} {
	interfaceVal := man.viper.Get(key)
	if interfaceVal == nil {
		var ok bool
		interfaceVal, ok = man.defaults[key]
		if !ok {
			panic("Tried to look up default value for nonexistent config option: " + key)
		}
	}
	return interfaceVal
}

// addConfigString adds a string config to the config options
func (man Manager) addConfigString(key, defVal, usage string) {
	man.command.PersistentFlags().String(flagNameFromConfigKey(key), defVal, getFlagUsage(key, usage))
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key)))
	man.viper.BindEnv(key, envNameFromConfigKey(key))

	// Add default
	man.addDefault(key, defVal)
}

// getConfigString retrieves a string from the loaded config
func (man Manager) getConfigString(key string) string {
	interfaceVal := man.getInterfaceVal(key)
	stringVal, err := cast.ToStringE(interfaceVal)
	if err != nil {
		panic("Unable to cast to string for key " + key + ": " + err.Error())
	}

	return stringVal
}

// Custom handling for TLSProfile which can only accept specific values
// for the argument
func (man Manager) getConfigTLSProfile() string {
	ival := man.getInterfaceVal(TLSProfileKey)
	sval, err := cast.ToStringE(ival)
	if err != nil {
		panic(fmt.Sprintf("%s requires a string value: %s", TLSProfileKey, err.Error()))
	}
	switch sval {
	case TLSProfileModern, TLSProfileIntermediate, TLSProfileOld:
	default:
		panic(fmt.Sprintf("%s must be one of %s, %s or %s", TLSProfileKey,
			TLSProfileModern, TLSProfileIntermediate, TLSProfileOld))
	}
	return sval
}

// addConfigInt adds a int config to the config options
func (man Manager) addConfigInt(key string, defVal int, usage string) {
	man.command.PersistentFlags().Int(flagNameFromConfigKey(key), defVal, getFlagUsage(key, usage))
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key)))
	man.viper.BindEnv(key, envNameFromConfigKey(key))

	// Add default
	man.addDefault(key, defVal)
}

// getConfigInt retrieves a int from the loaded config
func (man Manager) getConfigInt(key string) int {
	interfaceVal := man.getInterfaceVal(key)
	intVal, err := cast.ToIntE(interfaceVal)
	if err != nil {
		panic("Unable to cast to int for key " + key + ": " + err.Error())
	}

	return intVal
}

// addConfigBool adds a bool config to the config options
func (man Manager) addConfigBool(key string, defVal bool, usage string) {
	man.command.PersistentFlags().Bool(flagNameFromConfigKey(key), defVal, getFlagUsage(key, usage))
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key)))
	man.viper.BindEnv(key, envNameFromConfigKey(key))

	// Add default
	man.addDefault(key, defVal)
}

// getConfigBool retrieves a bool from the loaded config
func (man Manager) getConfigBool(key string) bool {
	interfaceVal := man.getInterfaceVal(key)
	boolVal, err := cast.ToBoolE(interfaceVal)
	if err != nil {
		panic("Unable to cast to bool for key " + key + ": " + err.Error())
	}

	return boolVal
}

// addConfigDuration adds a duration config to the config options
func (man Manager) addConfigDuration(key string, defVal time.Duration, usage string) {
	man.command.PersistentFlags().Duration(flagNameFromConfigKey(key), defVal, getFlagUsage(key, usage))
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key)))
	man.viper.BindEnv(key, envNameFromConfigKey(key))

	// Add default
	man.addDefault(key, defVal)
}

// getConfigDuration retrieves a duration from the loaded config
func (man Manager) getConfigDuration(key string) time.Duration {
	interfaceVal := man.getInterfaceVal(key)
	durationVal, err := cast.ToDurationE(interfaceVal)
	if err != nil {
		panic("Unable to cast to duration for key " + key + ": " + err.Error())
	}

	return durationVal
}

// loadConfigFile handles the loading of the config file.
func (man Manager) loadConfigFile() {
	man.viper.SetConfigType("yaml")

	configFile := man.command.PersistentFlags().Lookup("config").Value.String()

	if configFile == "" {
		// No config file set, only use configs from env
		// vars/flags/defaults
		return
	}

	man.viper.SetConfigFile(configFile)
	err := man.viper.ReadInConfig()

	fmt.Println("Using config file: ", man.viper.ConfigFileUsed())

	if err != nil {
		panic("Error reading config: " + err.Error())
	}
}

// TestConfig returns a barebones configuration suitable for use in tests.
// Individual tests may want to override some of the values provided.
func TestConfig() KolideConfig {
	return KolideConfig{
		App: AppConfig{
			TokenKey:                  "CHANGEME",
			InviteTokenValidityPeriod: 5 * 24 * time.Hour,
		},
		Auth: AuthConfig{
			JwtKey:      "CHANGEME",
			BcryptCost:  6, // Low cost keeps tests fast
			SaltKeySize: 24,
		},
		Session: SessionConfig{
			KeySize:  64,
			Duration: 24 * 90 * time.Hour,
		},
		Osquery: OsqueryConfig{
			NodeKeySize:         24,
			StatusLogFile:       "/dev/null",
			ResultLogFile:       "/dev/null",
			LabelUpdateInterval: 1 * time.Hour,
		},
		Logging: LoggingConfig{
			Debug:         true,
			DisableBanner: true,
		},
	}
}
