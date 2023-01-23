package table

import (
	"context"
	"math/rand"
	"testing"
	"time"

	osquery_client "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/gen/osquery"
	"github.com/osquery/osquery-go/mock"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBestPractices(t *testing.T) {
	t.Parallel()

	mock := &mock.ExtensionManager{}
	client := &osquery_client.ExtensionManagerClient{Client: mock}

	rand.Seed(time.Now().Unix())

	// Generate random fake query values
	expectedRow := map[string]string{}
	queryValues := map[string]string{}
	for col, query := range bestPracticesSimpleColumns {
		val := "0"
		if rand.Int()%2 == 0 {
			val = "1"
		}
		expectedRow[col] = val
		queryValues[query] = val
	}

	mock.QueryFunc = func(ctx context.Context, sql string) (*osquery.ExtensionResponse, error) {
		val, ok := queryValues[sql]
		if !ok {
			return &osquery.ExtensionResponse{
				Status: &osquery.ExtensionStatus{Code: 1, Message: "unknown query"},
			}, nil
		}
		return &osquery.ExtensionResponse{
			Status:   &osquery.ExtensionStatus{Code: 0, Message: "OK"},
			Response: []map[string]string{{"compliant": val}},
		}, nil
	}

	generateFunc := generateBestPractices(client)
	rows, err := generateFunc(context.Background(), table.QueryContext{})
	require.Nil(t, err)
	if assert.Equal(t, 1, len(rows)) {
		assert.Equal(t, expectedRow, rows[0])
	}
}
