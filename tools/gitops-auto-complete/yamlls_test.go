package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestYAMLLS validates the fixtures against a real yaml-language-server — the
// actual editor target — so the generated schema is checked with the same engine
// users see. It's gated behind YAMLLS_TEST=1 because it needs the yamlls binary
// (a node program). Set YAMLLS_BIN to point at the binary if it isn't on PATH.
//
// The schema is attached per-file with a modeline; only Error-severity (schema)
// diagnostics are counted, so a deprecation hint on a valid file doesn't fail it.
func TestYAMLLS(t *testing.T) {
	if os.Getenv("YAMLLS_TEST") == "" {
		t.Skip("set YAMLLS_TEST=1 to validate fixtures against a real yaml-language-server")
	}
	bin := os.Getenv("YAMLLS_BIN")
	if bin == "" {
		path, err := exec.LookPath("yaml-language-server")
		if err != nil {
			t.Fatal("YAMLLS_TEST is set but yaml-language-server is not on PATH; add it to PATH or set YAMLLS_BIN")
		}
		bin = path
	}
	schemaPath, err := filepath.Abs(schemaFile)
	if err != nil {
		t.Fatal(err)
	}

	yamlls := startYAMLLS(t, bin)
	defer yamlls.close()

	run := func(dir string, wantErrors bool) {
		files, _ := filepath.Glob(filepath.Join("testdata", dir, "*.yml"))
		for _, file := range files {
			t.Run(dir+"/"+filepath.Base(file), func(t *testing.T) {
				content, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				doc := "# yaml-language-server: $schema=" + schemaPath + "\n" + string(content)
				errs := yamlls.diagnose(t, doc)
				switch {
				case wantErrors && errs == 0:
					t.Errorf("%s: expected yamlls schema errors, got none", file)
				case !wantErrors && errs > 0:
					t.Errorf("%s: expected no yamlls schema errors, got %d", file, errs)
				}
			})
		}
	}
	run("valid", false)
	run("invalid", true)
}

type yamllsClient struct {
	cmd  *exec.Cmd
	in   io.WriteCloser
	mu   sync.Mutex // serializes writes (main + auto-responses from readLoop)
	msgs chan map[string]any
	id   int
	uri  int
}

func startYAMLLS(t *testing.T, bin string) *yamllsClient {
	t.Helper()
	cmd := exec.Command(bin, "--stdio")
	cmd.Stderr = os.Stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start yaml-language-server: %v", err)
	}
	c := &yamllsClient{cmd: cmd, in: in, msgs: make(chan map[string]any, 64)}
	go c.readLoop(out)

	c.request(t, "initialize", map[string]any{
		"processId": nil,
		"rootUri":   nil,
		"capabilities": map[string]any{
			"textDocument": map[string]any{"publishDiagnostics": map[string]any{}},
		},
	})
	c.notify(t, "initialized", map[string]any{})
	return c
}

func (c *yamllsClient) close() {
	_ = c.in.Close()
	_ = c.cmd.Process.Kill()
	_ = c.cmd.Wait()
}

func (c *yamllsClient) write(t *testing.T, m map[string]any) {
	t.Helper()
	b, _ := json.Marshal(m)
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := fmt.Fprintf(c.in, "Content-Length: %d\r\n\r\n%s", len(b), b); err != nil {
		t.Fatalf("write lsp message: %v", err)
	}
}

func (c *yamllsClient) notify(t *testing.T, method string, params any) {
	c.write(t, map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func (c *yamllsClient) request(t *testing.T, method string, params any) {
	t.Helper()
	c.id++
	id := c.id
	c.write(t, map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	timeout := time.After(15 * time.Second)
	for {
		select {
		case m, ok := <-c.msgs:
			if !ok {
				t.Fatalf("yamlls closed while waiting for %s", method)
			}
			// A response has an id and no method.
			if _, hasMethod := m["method"]; !hasMethod && idOf(m) == id {
				return
			}
		case <-timeout:
			t.Fatalf("timeout waiting for %s response", method)
		}
	}
}

// diagnose opens a fresh document and returns how many Error-severity diagnostics
// yamlls publishes for it.
func (c *yamllsClient) diagnose(t *testing.T, doc string) int {
	t.Helper()
	c.uri++
	uri := fmt.Sprintf("file:///tmp/gitops-schema-test-%d.yaml", c.uri)
	c.notify(t, "textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "yaml", "version": 1, "text": doc},
	})

	var latest []any
	found := false
	hard := time.After(15 * time.Second)
	for {
		var quiet <-chan time.Time
		if found {
			// yamlls may publish an empty set first, then the real one after the
			// schema loads; wait for a quiet period before finalizing.
			quiet = time.After(1 * time.Second)
		}
		select {
		case m, ok := <-c.msgs:
			if !ok {
				t.Fatal("yamlls closed while waiting for diagnostics")
			}
			if m["method"] == "textDocument/publishDiagnostics" {
				if p, _ := m["params"].(map[string]any); p != nil && p["uri"] == uri {
					latest, _ = p["diagnostics"].([]any)
					found = true
				}
			}
		case <-quiet:
			return countErrors(latest)
		case <-hard:
			if !found {
				t.Fatal("timed out waiting for yamlls diagnostics")
			}
			return countErrors(latest)
		}
	}
}

func (c *yamllsClient) readLoop(r io.Reader) {
	br := bufio.NewReader(r)
	for {
		length := 0
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				close(c.msgs)
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if strings.HasPrefix(strings.ToLower(line), "content-length:") {
				length, _ = strconv.Atoi(strings.TrimSpace(line[len("content-length:"):]))
			}
		}
		if length == 0 {
			continue
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(br, body); err != nil {
			close(c.msgs)
			return
		}
		var m map[string]any
		if json.Unmarshal(body, &m) != nil {
			continue
		}
		// Auto-respond to server->client requests (method + id) so yamlls doesn't
		// block; forward responses and notifications to the channel.
		if _, hasMethod := m["method"]; hasMethod {
			if _, hasID := m["id"]; hasID {
				c.respond(m)
				continue
			}
		}
		c.msgs <- m
	}
}

// respond returns a null (or empty-array for workspace/configuration) result to a
// server-initiated request.
func (c *yamllsClient) respond(req map[string]any) {
	var result any
	if req["method"] == "workspace/configuration" {
		items := 0
		if p, _ := req["params"].(map[string]any); p != nil {
			if arr, _ := p["items"].([]any); arr != nil {
				items = len(arr)
			}
		}
		result = make([]any, items)
	}
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": result})
	c.mu.Lock()
	defer c.mu.Unlock()
	_, _ = fmt.Fprintf(c.in, "Content-Length: %d\r\n\r\n%s", len(b), b)
}

func idOf(m map[string]any) int {
	if f, ok := m["id"].(float64); ok {
		return int(f)
	}
	return -1
}

func countErrors(diags []any) int {
	count := 0
	for _, diag := range diags {
		if fields, ok := diag.(map[string]any); ok {
			if sev, ok := fields["severity"].(float64); ok && sev == 1 {
				count++
			}
		}
	}
	return count
}
