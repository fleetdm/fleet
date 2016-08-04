package main

import (
	"encoding/json"
	"io/ioutil"
)

type mysqlConfigData struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type serverConfigData struct {
	Address string `json:"address"`
	Cert    string `json:"cert"`
	Key     string `json:"key"`
}

type appConfigData struct {
	BcryptCost               int     `json:"bcrypt_cost"`
	JWTKey                   string  `json:"jwt_key"`
	SaltKeySize              int     `json:"salt_key_size"`
	SessionKeySize           int     `json:"session_key_size"`
	SessionExpirationSeconds float64 `json:"session_expiration_seconds"`
}

type configData struct {
	MySQL  mysqlConfigData  `json:"mysql"`
	Server serverConfigData `json:"server"`
	App    appConfigData    `json:"app"`
}

var (
	config configData
)

var defaultMysqlConfigData = mysqlConfigData{
	Address:  "mysql:3306",
	Username: "kolide",
	Password: "kolide",
	Database: "kolide",
}

var defaultServerConfigData = serverConfigData{
	Address: "127.0.0.1:8080",
	Cert:    "./tools/kolide.crt",
	Key:     "./tools/kolide.key",
}

var defaultAppConfigData = appConfigData{
	BcryptCost:               12,
	JWTKey:                   "very secure",
	SessionKeySize:           64,
	SaltKeySize:              24,
	SessionExpirationSeconds: 60 * 60 * 24 * 90,
}

var defaultConfigData = configData{
	MySQL:  defaultMysqlConfigData,
	Server: defaultServerConfigData,
	App:    defaultAppConfigData,
}

func setDefaultConfigValues() {
	config = defaultConfigData
}

func loadConfig(path string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, &config)
	return err
}
