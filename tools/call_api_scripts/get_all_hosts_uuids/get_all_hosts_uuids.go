package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage 1: go run get_all_hosts_uuids <JWT_TOKEN>")
		fmt.Println("Usage 2: go run get_all_hosts_uuids.go <TOKEN> | grep uuid  |  awk '{ print  substr($2, 2, length($2)-3)  }'  > uuids_list.txt")
		return
	}
	jwtToken := os.Args[1]

	apiURL := "https://dogfood.fleetdm.com/api/v1/fleet/hosts"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
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
