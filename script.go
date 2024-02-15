package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	// Check if UUID is provided as a command-line argument
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <UUID> <JWT_TOKEN>")
		return
	}

	// Extract UUID and JWT token from command-line arguments
	uuid := os.Args[1]
	jwtToken := os.Args[2]

	// Define the request body
	requestBody := map[string]string{
		"query": "SELECT * FROM time;",
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("Error marshalling request body:", err)
		return
	}

	// API endpoint URL
	apiUrl := fmt.Sprintf("https://dogfood.fleetdm.com/api/v1/fleet/hosts/identifier/%s/query", uuid)

	// Create a new HTTP request
	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Set JWT token in the Authorization header
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request using default HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Print full response
	fmt.Println("Response:")
	fmt.Println(string(responseBody))
}
