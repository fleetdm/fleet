package service

import (
	"encoding/json"
	"net/http"

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
