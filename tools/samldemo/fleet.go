package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const fleetKey = "jdrS2C9j0TR2bbDPil3jywHwyq5uujyAT1idiymuJMoivsplD5fLRpsJkepLail4Zgfu5kIV8kshwjwou+tbiw=="
const fleetURL = "https://dogfood.fleetdm.com"

type translatePayload struct {
	Identifier string `json:"identifier"`
	ID         int    `json:"id,omitempty"`
}

type translateListEntry struct {
	Type    string           `json:"type"`
	Payload translatePayload `json:"payload"`
}

type translateRequestResponse struct {
	List []translateListEntry `json:"list"`
}

func hostIDsForUser(email string) ([]int, error) {
	b, err := json.Marshal(translateRequestResponse{
		List: []translateListEntry{
			{
				Type:    "host",
				Payload: translatePayload{Identifier: email},
			},
		},
	})
	log.Println(string(b))
	req, err := http.NewRequest("POST", fleetURL+"/api/v1/fleet/translate", bytes.NewBufferString(`{}`)) //bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+fleetKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)

	body, err := io.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}
	log.Println(string(body))

	var trr translateRequestResponse
	err = json.Unmarshal(body, &trr)
	if err != nil {
		return nil, err
	}

	log.Println(trr)

	res := []int{}
	for _, entry := range trr.List {
		res = append(res, entry.Payload.ID)
	}

	return res, nil
}

func checkForEmail(email string) error {

	return nil
}
