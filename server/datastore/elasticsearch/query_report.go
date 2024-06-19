package query_report

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

type OpenSearchService struct {
	Client      *opensearch.Client
	BulkIndexer *BulkIndexer
}

// BulkIndexer is a structure to handle bulk indexing
type BulkIndexer struct {
	client     *opensearch.Client
	batch      []string
	batchSize  int
	mutex      sync.Mutex
	flushTimer *time.Ticker
}

// NewBulkIndexer initializes a new BulkIndexer
func NewBulkIndexer(client *opensearch.Client, batchSize int, flushInterval time.Duration) *BulkIndexer {
	bi := &BulkIndexer{
		client:     client,
		batchSize:  batchSize,
		flushTimer: time.NewTicker(flushInterval),
	}

	go func() {
		for range bi.flushTimer.C {
			bi.Flush()
		}
	}()

	return bi
}

// Add adds a document to the batch
func (bi *BulkIndexer) Add(index string, documentID string, doc interface{}) error {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()

	meta := map[string]interface{}{
		"index": map[string]interface{}{
			"_index": index,
			"_id":    documentID,
		},
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	docJSON, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	bi.batch = append(bi.batch, string(metaJSON), string(docJSON))

	if len(bi.batch) >= bi.batchSize*2 {
		return bi.Flush()
	}

	return nil
}

// Flush sends the batched documents to OpenSearch
func (bi *BulkIndexer) Flush() error {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()

	if len(bi.batch) == 0 {
		return nil
	}

	body := strings.Join(bi.batch, "\n") + "\n"

	req := opensearchapi.BulkRequest{
		Body: strings.NewReader(body),
	}

	res, err := req.Do(context.Background(), bi.client)
	if err != nil {
		return fmt.Errorf("error executing bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response from bulk request: %s", res.String())
	}

	bi.batch = bi.batch[:0]
	return nil
}

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
			os.Getenv("FLEET_OPENSEARCH_ENDPOINT"),
		},
		Username: os.Getenv("FLEET_OPENSEARCH_USERNAME"),
		Password: os.Getenv("FLEET_OPENSEARCH_PASSWORD"),
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

	log.Println("OpenSearch Index created")
	return nil
}

// UpsertHostSnapshot inserts or updates a document in the OpenSearch index
func UpsertHostSnapshot(bulkIndexer *BulkIndexer, result fleet.ScheduledQueryResult) error {
	doc := map[string]interface{}{
		"hostIdentifier": result.OsqueryHostID,
		"name":           result.QueryName,
		"snapshot":       result.Snapshot,
		"unixTime":       result.UnixTime,
	}

	documentID := url.QueryEscape(result.OsqueryHostID + "-" + result.QueryName)

	return bulkIndexer.Add("report", documentID, doc)
}
