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
	BcryptCost int    `json:"bcrypt_cost"`
	SaltLength int    `json:"salt_length"`
	JWTKey     string `json:"jwt_key"`
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
	Address:  "127.0.0.1:3306",
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
	BcryptCost: 12,
	SaltLength: 32,
	JWTKey:     "very secure",
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
