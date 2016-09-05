package config

import (
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// File may or may not contain the path to the config file
	File string
)

func init() {
	cobra.OnInitialize(initConfig)
}

// Due to a deficiency in viper (https://github.com/spf13/viper/issues/71), one
// can not set the default values of nested config elements. For example, if the
// "mysql" section of the config allows a user to define "username", "password",
// and "database", but the only wants to override the default for "username".
// they should be able to create a config which looks like:
//
//   mysql:
//     username: foobar
//
// In viper, that would nullify the default values of all other config keys in
// the mysql section ("mysql.*"). To get around this, instead of using the
// provided API for setting default values, after we've read the config and env,
// we manually check to see if the value has been set and, if it hasn't, we set
// it manually.
func setDefaultConfigValue(key string, value interface{}) {
	if viper.Get(key) == nil {
		viper.Set(key, value)
	}
}

func initConfig() {
	if File != "" {
		viper.SetConfigFile(File)
	}
	viper.SetConfigName("kolide")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath("./tools/app")
	viper.AddConfigPath("/etc/kolide")

	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("KOLIDE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		logrus.Infoln("Not reading config file. Relying on environment variables and default values.")
	}

	setDefaultConfigValue("mysql.address", "localhost:3306")
	setDefaultConfigValue("mysql.username", "kolide")
	setDefaultConfigValue("mysql.password", "kolide")
	setDefaultConfigValue("mysql.database", "kolide")

	setDefaultConfigValue("server.address", "0.0.0.0:8080")

	setDefaultConfigValue("app.web_address", "0.0.0.0:8080")

	setDefaultConfigValue("auth.jwt_key", "CHANGEME")
	setDefaultConfigValue("auth.bcrypt_cost", 12)
	setDefaultConfigValue("auth.salt_key_size", 24)

	setDefaultConfigValue("smtp.token_key_size", 24)
	setDefaultConfigValue("smtp.address", "localhost:1025")
	setDefaultConfigValue("smtp.pool_connections", 4)

	setDefaultConfigValue("session.key_size", 64)
	setDefaultConfigValue("session.expiration_seconds", 60*60*24*90)
	setDefaultConfigValue("session.cookie_name", "KolideSession")

	setDefaultConfigValue("osquery.node_key_size", 24)
	setDefaultConfigValue("osquery.status_log_file", "/tmp/osquery_status")
	setDefaultConfigValue("osquery.result_log_file", "/tmp/osquery_result")
	setDefaultConfigValue("osquery.label_up_interval", 1*time.Minute)

	setDefaultConfigValue("logging.debug", false)
	setDefaultConfigValue("logging.disable_banner", false)

	if viper.GetBool("logging.debug") {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	if viper.GetBool("logs.json") {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
}
