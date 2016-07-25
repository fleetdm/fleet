package main

import (
	"encoding/json"
	"io/ioutil"
)

type configData struct {
	MySQL struct {
		Address  string `json:"address"`
		Username string `json:"username"`
		Password string `json:"password"`
		Database string `json:"database"`
	} `json:"mysql"`
	Server struct {
		Address string `json:"address"`
		Cert    string `json:"cert"`
		Key     string `json:"key"`
	} `json:"server"`
}

var (
	config configData
)

func loadConfig(path string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, &config)
	return err
}
