package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ListCarves lists the file carving sessions
func (c *Client) ListCarves(opt fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
	endpoint := "/api/v1/fleet/carves"
	rawQuery := ""
	if opt.Expired {
		rawQuery = "expired=1"
	}
	response, err := c.AuthenticatedDo("GET", endpoint, rawQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/carves: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"list carves received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody listCarvesResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get carves response: %w", err)
	}
	if responseBody.Err != nil {
		return nil, fmt.Errorf("get carves: %s", responseBody.Err)
	}

	carves := []*fleet.CarveMetadata{}
	for _, carve := range responseBody.Carves {
		c := carve
		carves = append(carves, &c)
	}

	return carves, nil
}

func (c *Client) GetCarve(carveId int64) (*fleet.CarveMetadata, error) {
	endpoint := fmt.Sprintf("/api/v1/fleet/carves/%d", carveId)
	response, err := c.AuthenticatedDo("GET", endpoint, "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET "+endpoint+": %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get carve received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}
	var responseBody getCarveResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode carve response: %w", err)
	}
	if responseBody.Err != nil {
		return nil, fmt.Errorf("get carve: %s", responseBody.Err)
	}

	return &responseBody.Carve, nil
}

func (c *Client) getCarveBlock(carveId, blockId int64) ([]byte, error) {
	path := fmt.Sprintf(
		"/api/v1/fleet/carves/%d/block/%d",
		carveId,
		blockId,
	)
	response, err := c.AuthenticatedDo("GET", path, "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get carve block received status %d: %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getCarveBlockResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get carve block response: %w", err)
	}
	if responseBody.Err != nil {
		return nil, fmt.Errorf("get carve block: %s", responseBody.Err)
	}

	return responseBody.Data, nil
}

type carveReader struct {
	carve     fleet.CarveMetadata
	bytesRead int64
	curBlock  int64
	buffer    []byte
	client    *Client
}

func newCarveReader(carve fleet.CarveMetadata, client *Client) *carveReader {
	return &carveReader{
		carve:     carve,
		client:    client,
		bytesRead: 0,
		curBlock:  0,
	}
}

func (r *carveReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.bytesRead >= r.carve.CarveSize {
		return 0, io.EOF
	}

	// Load data from API if necessary
	if len(r.buffer) == 0 {
		var err error
		r.buffer, err = r.client.getCarveBlock(r.carve.ID, r.curBlock)
		if err != nil {
			return 0, fmt.Errorf("get block %d: %w", r.curBlock, err)
		}
		r.curBlock++
	}

	// Calculate length we can copy
	copyLen := len(p)
	if copyLen > len(r.buffer) {
		copyLen = len(r.buffer)
	}

	// Perform copy and clear copied contents from buffer
	copy(p, r.buffer[:copyLen])
	r.buffer = r.buffer[copyLen:]

	r.bytesRead += int64(copyLen)

	return copyLen, nil
}

// DownloadCarve creates a Reader downloading a carve (by ID)
func (c *Client) DownloadCarve(id int64) (io.Reader, error) {
	path := fmt.Sprintf("/api/v1/fleet/carves/%d", id)
	response, err := c.AuthenticatedDo("GET", path, "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"download carve received status %d: %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getCarveResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get carve by name response: %w", err)
	}
	if responseBody.Err != nil {
		return nil, fmt.Errorf("get carve by name: %s", responseBody.Err)
	}

	reader := newCarveReader(responseBody.Carve, c)

	return reader, nil
}
