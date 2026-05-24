package command

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

// Client is a thin Fleet API client. Bearer auth, JSON in/out.
type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	dryRun     bool
	verbose    bool
}

func newClientFromViper() (*Client, error) {
	raw := viper.GetString(keyFleetURL)
	if raw == "" {
		return nil, fmt.Errorf("fleet-url is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid fleet-url %q: %w", raw, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("fleet-url must include scheme and host, got %q", raw)
	}
	httpClient := &http.Client{Timeout: 60 * time.Second}
	if viper.GetBool(keyInsecure) {
		// Local dev Fleets typically use a self-signed cert. Honor --insecure
		// the same way fleetctl does.
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // opt-in dev flag
		}
	}
	return &Client{
		baseURL:    u,
		token:      viper.GetString(keyAPIToken),
		httpClient: httpClient,
		dryRun:     viper.GetBool("dry-run"),
		verbose:    viper.GetBool("verbose"),
	}, nil
}

// AlreadyExistsError signals that the resource conflicted with an existing one.
// Seeders treat this as a soft success (idempotent) unless --strict is added later.
type AlreadyExistsError struct{ Msg string }

func (e *AlreadyExistsError) Error() string         { return e.Msg }
func (e *AlreadyExistsError) IsAlreadyExists() bool { return true }

func (c *Client) url(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(c.baseURL.String(), "/") + path
}

// do performs an HTTP request and decodes the JSON response into out (if non-nil).
func (c *Client) do(method, path string, body io.Reader, contentType string, out any) error {
	if c.dryRun {
		fmt.Printf("[dry-run] %s %s\n", method, path)
		return nil
	}
	req, err := http.NewRequest(method, c.url(path), body)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	if c.verbose {
		fmt.Fprintf(stderrish, "→ %s %s\n", method, req.URL.Path)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusConflict || (resp.StatusCode == http.StatusUnprocessableEntity && bytes.Contains(rb, []byte("already exists"))) {
		return &AlreadyExistsError{Msg: strings.TrimSpace(string(rb))}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s → %d: %s", method, req.URL.Path, resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	if out != nil && len(rb) > 0 {
		if err := json.Unmarshal(rb, out); err != nil {
			return fmt.Errorf("decode response: %w (body: %s)", err, string(rb))
		}
	}
	return nil
}

func (c *Client) Get(path string, out any) error {
	return c.do(http.MethodGet, path, nil, "", out)
}

func (c *Client) Post(path string, body any, out any) error {
	r, err := jsonBody(body)
	if err != nil {
		return err
	}
	return c.do(http.MethodPost, path, r, "application/json", out)
}

func (c *Client) Patch(path string, body any, out any) error {
	r, err := jsonBody(body)
	if err != nil {
		return err
	}
	return c.do(http.MethodPatch, path, r, "application/json", out)
}

func (c *Client) Delete(path string) error {
	return c.do(http.MethodDelete, path, nil, "", nil)
}

// PostMultipart uploads form fields + named files. Used for script / profile / installer uploads.
// The Fleet API distinguishes the form *field name* (e.g. "script", "profile")
// from the *filename*, which the server inspects for extension-based platform
// detection — so we carry both via seed.MultipartFile.
func (c *Client) PostMultipart(path string, fields map[string]string, files []seed.MultipartFile, out any) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	for _, f := range files {
		fw, err := w.CreateFormFile(f.FieldName, f.Filename)
		if err != nil {
			return err
		}
		if _, err := fw.Write(f.Content); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.do(http.MethodPost, path, &buf, w.FormDataContentType(), out)
}

func jsonBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}
