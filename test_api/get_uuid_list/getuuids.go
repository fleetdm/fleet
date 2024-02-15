package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// HostResponse represents the structure of the JSON response containing host IDs

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run getuuids.go <JWT_TOKEN>")
		return
	}
	jwtToken := os.Args[1]

	apiURL := "https://dogfood.fleetdm.com/api/v1/fleet/hosts"

	// Create a new HTTP request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Set JWT token in the Authorization header
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	// Send the request using default HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(responseBody))
}
