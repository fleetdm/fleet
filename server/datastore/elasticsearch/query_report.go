package query_report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

// CreateOpenSearchClient initializes an OpenSearch client using the default credential provider chain
func CreateOpenSearchClient() (*opensearch.Client, error) {
	// Create a new session using default credentials (IAM role, environment variables, etc.)
	_, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"), // Change to your region
	})
	if err != nil {
		return nil, fmt.Errorf("error creating AWS session: %w", err)
	}

	// OpenSearch configuration
	cfg := opensearch.Config{
		Addresses: []string{
			os.Getenv("FLEET_OPENSEARCH_ENDPOINT"), // Change to your OpenSearch domain endpoint
		},
	}

	es, err := opensearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating OpenSearch client: %w", err)
	}
	return es, nil
}

// CreateIndex creates the index if it doesn't exist
func CreateIndex(client *opensearch.Client) error {
	indexName := "report"

	// Check if the index exists
	req := opensearchapi.IndicesExistsRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(context.Background(), client)
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

	createReq := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mapping),
	}

	createRes, err := createReq.Do(context.Background(), client)
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

// UpsertHostSnapshot inserts or updates a document in the OpenSearch index
func UpsertHostSnapshot(client *opensearch.Client, result fleet.ScheduledQueryResult) error {
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

	req := opensearchapi.IndexRequest{
		Index:      "report",
		DocumentID: documentID,
		Body:       bytes.NewReader(bodyJSON),
		Refresh:    "true", // Optional: can be "true", "false", or "wait_for"
	}

	res, err := req.Do(ctx, client)
	if err != nil {
		return fmt.Errorf("error executing index request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response from index request: %s", res.String())
	}

	return nil
}
