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
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run run_query.go <UUID> <JWT_TOKEN>")
		return
	}
	uuid := os.Args[1]
	jwtToken := os.Args[2]

	requestBody := map[string]string{
		"query": "SELECT * FROM time;",
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("Error marshalling request body:", err)
		return
	}

	apiUrl := fmt.Sprintf("https://dogfood.fleetdm.com/api/v1/fleet/hosts/identifier/%s/query", uuid)
	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

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
