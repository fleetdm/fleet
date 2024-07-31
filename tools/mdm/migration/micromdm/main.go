package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

var (
	apiToken = flag.String("api-token", "", "API token for the MicroMDM instance")
	url      = flag.String("url", "", "URL of the MicroMDM instance")
	port     = flag.String("port", "4648", "Port used by the webserver")
)

func main() {
	flag.Parse()

	if *apiToken == "" || *url == "" {
		log.Fatal("--api-token and --url are required.")
	}

	log.Println("getting API token...")
	client := newMicroMDMClient(*apiToken, *url)

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			log.Printf("ERROR: reading request body: %s\n", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(body) == 0 {
			log.Print("ERROR: empty request body")
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		var deviceInfo struct {
			Host struct {
				HardwareSerial string `json:"hardware_serial"`
			} `json:"host"`
		}
		if err := json.Unmarshal(body, &deviceInfo); err != nil {
			log.Printf("ERROR: unmarshalling request body: %s\n", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Printf("unenrolling %s", deviceInfo.Host.HardwareSerial)

		jamfID, err := client.getJamfID(deviceInfo.Host.HardwareSerial)
		if err != nil {
			log.Printf("ERROR: getting Jamf ID from serial number: %s\n", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Println("attempting to remove device from Jamf management...")
		if err := client.unmanageDevice(jamfID); err != nil {
			log.Printf("ERROR: unmanaging device: %s\n", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Printf("%s unenrolled", deviceInfo.Host.HardwareSerial)
	})

	log.Printf("server running at http://localhost:%s\n", *port)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", *port),
		ReadHeaderTimeout: 3 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf(err.Error())
	}
}

type microMDMClient struct {
	url   string
	token string
}

func newMicroMDMClient(apiToken, url string) *microMDMClient {
	client := &microMDMClient{url: url, token: apiToken}
	return client
}

func (m *microMDMClient) doWithRequest(req *http.Request) ([]byte, error) {
	client := fleethttp.NewClient()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		return body, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return body, nil
}

func (m *microMDMClient) do(method, path string) ([]byte, error) {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("accept", "application/xml")
	req.Header.Add("Authorization", "Bearer "+m.token)
	return m.doWithRequest(req)
}

func (m *microMDMClient) unmanageDevice(jamfID string) error {
	_, err := m.do("POST", fmt.Sprintf("%s/JSSResource/computercommands/command/UnmanageDevice/id/%s", *url, jamfID))
	return err
}

func (m *microMDMClient) getJamfID(serial string) (string, error) {
	body, err := m.do("GET", fmt.Sprintf("%s/JSSResource/computers/serialnumber/%s", *url, serial))
	if err != nil {
		return "", err
	}

	var data struct {
		XMLName xml.Name `xml:"computer"`
		ID      string   `xml:"general>id"`
	}

	if err := xml.Unmarshal(body, &data); err != nil {
		return "", err
	}

	return data.ID, nil
}
