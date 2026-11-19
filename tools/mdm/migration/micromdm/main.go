package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
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

	logger := slog.Default()
	client := newMicroMDMClient(*apiToken, *url)

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		body, err := io.ReadAll(request.Body)
		if err != nil {
			logger.With("error", err).ErrorContext(ctx, "reading request body")
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(body) == 0 {
			logger.ErrorContext(ctx, "empty request body")
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		logger.With("raw_body", string(body)).DebugContext(ctx, "got request")

		var deviceInfo struct {
			Host struct {
				UUID string `json:"uuid"`
			} `json:"host"`
		}
		if err := json.Unmarshal(body, &deviceInfo); err != nil {
			logger.With("device_uuid", deviceInfo.Host.UUID, "error", err).ErrorContext(ctx, "failed to unmarshal request body")
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		logger.With("device_uuid", deviceInfo.Host.UUID).InfoContext(ctx, "attempting to unenroll from MicroMDM")
		if err := client.unmanageDevice(deviceInfo.Host.UUID); err != nil {
			logger.With("device_uuid", deviceInfo.Host.UUID, "error", err).ErrorContext(ctx, "failed to unenroll device")
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		logger.With("device_uuid", deviceInfo.Host.UUID).InfoContext(ctx, "device unenrolled")
	})

	ctx := context.Background()
	logger.With("address", fmt.Sprintf("http://localhost:%s", *port)).InfoContext(ctx, "server running")
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", *port),
		ReadHeaderTimeout: 3 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
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

func (m *microMDMClient) do(method, path string, data any) ([]byte, error) {
	var body []byte
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		body = b
	}

	makeReq := func() (*http.Request, error) {
		if len(body) > 0 {
			return http.NewRequest(method, path, bytes.NewBuffer(body))
		}

		return http.NewRequest(method, path, nil)
	}

	req, err := makeReq()
	if err != nil {
		return nil, err
	}
	req.Header.Add("accept", "application/json")
	req.SetBasicAuth("micromdm", m.token)
	return m.doWithRequest(req)
}

func (m *microMDMClient) unmanageDevice(uuid string) error {
	req := struct {
		RequestType string `json:"request_type"`
		UDID        string `json:"udid"`
		Identifier  string `json:"identifier"`
	}{
		RequestType: "RemoveProfile",
		UDID:        uuid,
		Identifier:  "com.github.micromdm.micromdm.enroll",
	}
	_, err := m.do("POST", fmt.Sprintf("%s/v1/commands", m.url), &req)
	return err
}
