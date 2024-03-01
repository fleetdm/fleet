package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
)

const fleetURL = "https://dogfood.fleetdm.com"

var fleetKey = os.Getenv("FLEET_KEY")

type basicHost struct {
	ID       int    `json:"id"`
	Hostname string `json:"hostname"`
	Issues   struct {
		TotalIssuesCount     int `json:"total_issues_count"`
		FailingPoliciesCount int `json:"failing_policies_count"`
	} `json:"issues"`
}

func hostsForUser(email string) ([]basicHost, error) {
	req, err := http.NewRequest("GET", fleetURL+"/api/latest/fleet/hosts?page=0&per_page=50&query="+email, &bytes.Buffer{})
	req.Header.Set("Authorization", "Bearer "+fleetKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// log.Println("body: ", string(body))

	var hosts struct{ Hosts []basicHost }
	err = json.Unmarshal(body, &hosts)
	if err != nil {
		return nil, err
	}

	return hosts.Hosts, nil
}

func checkForEmail(email string) error {
	hosts, err := hostsForUser(email)
	if err != nil {
		return err
	}

	for _, host := range hosts {
		if host.Issues.FailingPoliciesCount > 0 {
			return errors.New("host is failing policies")
		}
	}

	if len(hosts) == 0 {
		log.Printf("No hosts found for %s", email)
	}

	return nil
}
