package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

// ListCarves lists the file carving sessions
func (c *Client) ListCarves(opt kolide.ListOptions) ([]*kolide.CarveMetadata, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/kolide/carves", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GET /api/v1/kolide/carves")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"list carves received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody listCarvesResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode get carves response")
	}
	if responseBody.Err != nil {
		return nil, errors.Errorf("get carves: %s", responseBody.Err)
	}

	carves := []*kolide.CarveMetadata{}
	for _, carve := range responseBody.Carves {
		c := carve.CarveMetadata
		carves = append(carves, &c)
	}

	return carves, nil
}

func (c *Client) getCarveBlock(name string, blockId int64) ([]byte, error) {
	path := fmt.Sprintf(
		"/api/v1/kolide/carves/%s/block/%d",
		url.PathEscape(name),
		blockId,
	)
	response, err := c.AuthenticatedDo("GET", path, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", path)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"get carve block received status %d: %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getCarveBlockResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode get carve block response")
	}
	if responseBody.Err != nil {
		return nil, errors.Errorf("get carve block: %s", responseBody.Err)
	}

	return responseBody.Data, nil
}

type carveReader struct {
	carve kolide.CarveMetadata
	bytesRead int64
	curBlock int64
	buffer []byte
	client *Client
}

func newCarveReader(carve kolide.CarveMetadata, client *Client) *carveReader {
	return &carveReader{
		carve:  carve,
		client: client,
		bytesRead: 0,
		curBlock:  0,
	}
}

func (r *carveReader)  Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.bytesRead >= r.carve.CarveSize {
		return 0, io.EOF
	}

	// Load data from API if necessary
	if len(r.buffer) == 0 {
		var err error
		r.buffer, err = r.client.getCarveBlock(r.carve.Name, r.curBlock)
		if err != nil {
			return 0, errors.Wrapf(err, "get block %d", r.curBlock)
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

// ListCarves lists the file carving sessio
func (c *Client) DownloadCarve(name string) (io.Reader, error) {
	path := fmt.Sprintf("/api/v1/kolide/carves/%s", name)
	response, err := c.AuthenticatedDo("GET", path, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", path)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"download carve received status %d: %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getCarveResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode get carve by name response")
	}
	if responseBody.Err != nil {
		return nil, errors.Errorf("get carve by name: %s", responseBody.Err)
	}

	reader := newCarveReader(responseBody.Carve.CarveMetadata, c)

	return reader, nil
}
