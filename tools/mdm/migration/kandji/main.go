package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

var (
	subdomainFlag = flag.String("subdomain", "", "Account subdomain")
	apiTokenFlag  = flag.String("api-token", "", "API token")
)

func main() {
	flag.Parse()

	if *subdomainFlag == "" || *apiTokenFlag == "" {
		log.Fatal("both --subdomain and --api-token must be provided")
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			fmt.Printf("ERROR: reading request body: %s\n", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(body) == 0 {
			fmt.Println("ERROR: empty request body")
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		var deviceInfo struct {
			Host struct {
				HardwareSerial string `json:"hardware_serial"`
			} `json:"host"`
		}
		if err := json.Unmarshal(body, &deviceInfo); err != nil {
			fmt.Printf("ERROR: unmarshalling request body: %s\n", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		fmt.Printf("unenrolling %s", deviceInfo.Host.HardwareSerial)
		if err := unenroll(deviceInfo.Host.HardwareSerial); err != nil {
			fmt.Printf("ERROR: unenrolling: %s\n", err)
			writer.WriteHeader(http.StatusBadGateway)
			return
		}

		fmt.Printf("successfully unenrolled device %s\n", deviceInfo.Host.HardwareSerial)
		writer.WriteHeader(http.StatusNoContent)
	})

	port := ":4648"
	if p := os.Getenv("SERVER_PORT"); p != "" {
		port = p
	}

	fmt.Printf("Server running at http://localhost%s\n", port)
	server := &http.Server{
		Addr:              port,
		ReadHeaderTimeout: 3 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
}

func unenroll(serialNumber string) error {
	if serialNumber == "" {
		return errors.New("empty serial number")
	}
	// Get device info to grab the device id
	// https://api-docs.kandji.io/#78209960-31a7-4e3b-a2c0-95c7e65bb5f9
	client := fleethttp.NewClient()
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s.api.kandji.io/api/v1/devices?serial_number=%s", *subdomainFlag, serialNumber), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *apiTokenFlag))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var deviceInfo []struct {
		DeviceID string `json:"device_id"`
	}
	if err = json.Unmarshal(bodyText, &deviceInfo); err != nil {
		return err
	}
	if len(deviceInfo) == 0 {
		return fmt.Errorf("empty deviceInfo response, serial: %s", serialNumber)
	}

	// Delete the device, which triggers unenrollment
	// https://api-docs.kandji.io/#97deb582-d86c-444a-aa3b-3528b9a8478f
	req, err = http.NewRequest("DELETE", fmt.Sprintf("https://%s.api.kandji.io/api/v1/devices/%s", *subdomainFlag, deviceInfo[0].DeviceID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *apiTokenFlag))
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	fmt.Println("resp.StatusCode, serialNumber, device", resp.StatusCode, serialNumber, deviceInfo[0].DeviceID)

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("received unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
