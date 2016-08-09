package config

import (
	"encoding/json"
	"io/ioutil"
)

type MySQLConfigData struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type ServerConfigData struct {
	Address string `json:"address"`
	Cert    string `json:"cert"`
	Key     string `json:"key"`
}

type AppConfigData struct {
	BcryptCost               int     `json:"bcrypt_cost"`
	Debug                    bool    `json:"debug"`
	JWTKey                   string  `json:"jwt_key"`
	SaltKeySize              int     `json:"salt_key_size"`
	SessionKeySize           int     `json:"session_key_size"`
	SessionExpirationSeconds float64 `json:"session_expiration_seconds"`
}

type configData struct {
	MySQL  MySQLConfigData  `json:"mysql"`
	Server ServerConfigData `json:"server"`
	App    AppConfigData    `json:"app"`
}

var defaultMySQLConfigData = MySQLConfigData{
	Address:  "mysql:3306",
	Username: "kolide",
	Password: "kolide",
	Database: "kolide",
}

var defaultServerConfigData = ServerConfigData{
	Address: "127.0.0.1:8080",
	Cert:    "./tools/osquery/kolide.crt",
	Key:     "./tools/osquery/kolide.key",
}

var defaultAppConfigData = AppConfigData{
	BcryptCost:               12,
	Debug:                    false,
	JWTKey:                   "very secure",
	SessionKeySize:           64,
	SaltKeySize:              24,
	SessionExpirationSeconds: 60 * 60 * 24 * 90,
}

var defaultConfigData = configData{
	MySQL:  defaultMySQLConfigData,
	Server: defaultServerConfigData,
	App:    defaultAppConfigData,
}

var (
	MySQL  MySQLConfigData
	Server ServerConfigData
	App    AppConfigData
)

func init() {
	MySQL = defaultMySQLConfigData
	Server = defaultServerConfigData
	App = defaultAppConfigData
}

func LoadConfig(path string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	var config configData
	err = json.Unmarshal(content, &config)
	if err != nil {
		return err
	}
	MySQL = config.MySQL
	App = config.App
	Server = config.Server
	return nil
}
