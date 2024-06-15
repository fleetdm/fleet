package query_report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func CreateIndex(es *elasticsearch.Client) error {
	// Check if the index exists
	req := esapi.IndicesExistsRequest{
		Index: []string{"hosts"},
	}

	res, err := req.Do(context.Background(), es)
	if err != nil {
		return fmt.Errorf("error checking if index exists: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		log.Println("Index already exists")
		return nil
	}

	// Create the index if it does not exist
	mapping := `{
		"mappings": {
			"properties": {
				"hostIdentifier": {"type": "keyword"},
				"name": {"type": "keyword"},
				"unixTime": {"type": "date"},
				"snapshot": {
					"type": "nested",
					"dynamic": true
				}
			}
		}
	}`

	createReq := esapi.IndicesCreateRequest{
		Index: "hosts",
		Body:  strings.NewReader(mapping),
	}

	createRes, err := createReq.Do(context.Background(), es)
	if err != nil {
		return fmt.Errorf("error creating index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("error response creating index: %s", createRes.String())
	}

	log.Println("Index created")
	return nil
}

func UpsertHostSnapshot(es *elasticsearch.Client, result fleet.ScheduledQueryResult) error {
	ctx := context.Background()

	// Document body for upsert
	doc := map[string]interface{}{
		"hostIdentifier": result.OsqueryHostID,
		"name":           result.QueryName,
		"snapshot":       result.Snapshot,
		"unixTime":       result.UnixTime,
	}

	bodyJSON, _ := json.Marshal(doc)

	documentID := url.QueryEscape(result.OsqueryHostID + "-" + result.QueryName)

	req := esapi.IndexRequest{
		Index:      "hosts",
		DocumentID: documentID,
		Body:       bytes.NewReader(bodyJSON),
	}

	res, err := req.Do(ctx, es)
	if err != nil {
		return fmt.Errorf("error executing index request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response from index request: %s", res.String())
	}

	return nil
}
