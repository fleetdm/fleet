package externalsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

const (
	linearAPIURL          = "https://api.linear.app/graphql"
	linearRequestTimeout  = 30 * time.Second
	linearMaxResponseSize = 1 << 20
)

// Linear is a Linear client to be used to make requests to the Linear external service.
type Linear struct {
	client *http.Client
	opts   LinearOptions
}

// LinearOptions defines the options to configure a Linear client.
type LinearOptions struct {
	APIKey string
	TeamID string
	// BaseURL overrides the Linear GraphQL endpoint. It is empty in production.
	BaseURL string
}

// LinearTeam is the subset of a Linear team returned by GetTeam.
type LinearTeam struct {
	ID string `json:"id"`
}

// NewLinearClient returns a Linear client to use to make requests to the Linear external service.
func NewLinearClient(opts *LinearOptions) *Linear {
	return &Linear{
		client: fleethttp.NewClient(fleethttp.WithTimeout(linearRequestTimeout)),
		opts:   *opts,
	}
}

// GetTeam returns the team for the team ID provided in the Linear client
// options. It can be used to test in one request the API key as well as the
// existence of the team.
func (l *Linear) GetTeam(ctx context.Context) (*LinearTeam, error) {
	var resp struct {
		Data struct {
			Team *LinearTeam `json:"team"` //nolint:apiparamcheck // Linear API response field, not a Fleet team
		} `json:"data"`
	}
	err := l.do(ctx, `query($id: String!) { team(id: $id) { id } }`, map[string]any{"id": l.opts.TeamID}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Data.Team == nil {
		return nil, errors.New("team not found")
	}
	return resp.Data.Team, nil
}

func (l *Linear) do(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := l.opts.BaseURL
	if url == "" {
		url = linearAPIURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Linear personal API keys are sent as-is, without a "Bearer" prefix.
	req.Header.Set("Authorization", l.opts.APIKey)

	res, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, linearMaxResponseSize))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Linear reports errors both as non-2xx statuses and as 200 responses with an errors array.
	var gqlErrs struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	_ = json.Unmarshal(raw, &gqlErrs)
	if len(gqlErrs.Errors) > 0 {
		msgs := make([]string, 0, len(gqlErrs.Errors))
		for _, e := range gqlErrs.Errors {
			msgs = append(msgs, e.Message)
		}
		return fmt.Errorf("status %d: %s", res.StatusCode, strings.Join(msgs, "; "))
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
