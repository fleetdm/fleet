// Package stepca provides test helpers for running a local step-ca ACME server.
//
// It manages the lifecycle of a step-ca Docker container with an ACME provisioner
// configured, suitable for integration testing of the ACME proxy.
package stepca

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Server represents a running step-ca instance for testing.
type Server struct {
	port          int
	containerName string
	rootCACert    []byte
	httpClient    *http.Client
}

// New creates and starts a step-ca server on a random available port.
// It registers cleanup to stop the container when the test finishes.
func New(t *testing.T) *Server {
	t.Helper()

	if _, ok := os.LookupEnv("ACME_TEST"); !ok {
		t.Skip("set ACME_TEST=1 to run ACME integration tests")
	}

	// Ensure docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatalf("docker not found in PATH: %v", err)
	}

	port := freePort(t)
	containerName := fmt.Sprintf("fleet-acme-test-stepca-%d", port)

	s := &Server{
		port:          port,
		containerName: containerName,
	}

	s.start(t)

	t.Cleanup(func() {
		s.stop(t)
	})

	return s
}

// DirectoryURL returns the ACME directory URL for this step-ca instance.
func (s *Server) DirectoryURL() string {
	return fmt.Sprintf("https://localhost:%d/acme/acme/directory", s.port)
}

// Port returns the port the step-ca instance is listening on.
func (s *Server) Port() int {
	return s.port
}

// RootCACert returns the PEM-encoded root CA certificate.
func (s *Server) RootCACert() []byte {
	return s.rootCACert
}

// TLSConfig returns a *tls.Config that trusts this step-ca instance.
func (s *Server) TLSConfig() *tls.Config {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(s.rootCACert)
	return &tls.Config{
		RootCAs: pool,
	}
}

// HTTPClient returns an *http.Client configured to trust this step-ca instance.
func (s *Server) HTTPClient() *http.Client {
	if s.httpClient == nil {
		s.httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: s.TLSConfig(),
			},
			Timeout: 30 * time.Second,
		}
	}
	return s.httpClient
}

func (s *Server) start(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Run step-ca in a single container with init script
	// First, create a volume for persistence
	volumeName := s.containerName + "-vol"
	run(t, ctx, "docker", "volume", "create", volumeName)

	// Initialize the CA
	initScript := `
set -e
if [ ! -f /home/step/config/ca.json ]; then
  mkdir -p /home/step/secrets
  echo "testpassword" > /home/step/secrets/password
  step ca init \
    --name="Fleet ACME Test CA" \
    --provisioner="admin" \
    --dns="localhost" \
    --address=":9443" \
    --password-file=/home/step/secrets/password
  step ca provisioner add acme --type ACME
fi
`
	run(t, ctx, "docker", "run", "--rm",
		"--name", s.containerName+"-init",
		"-v", volumeName+":/home/step",
		"--entrypoint", "/bin/sh",
		"smallstep/step-ca:latest",
		"-c", initScript,
	)

	// Extract root CA cert before starting the server
	rootCert := runOutput(t, ctx, "docker", "run", "--rm",
		"-v", volumeName+":/home/step",
		"--entrypoint", "cat",
		"smallstep/step-ca:latest",
		"/home/step/certs/root_ca.crt",
	)
	s.rootCACert = []byte(rootCert)

	// Patch the CA config to listen on our chosen port instead of 9443,
	// and use host networking so step-ca can reach challenge servers on localhost.
	patchScript := fmt.Sprintf(`sed -i 's/:9443/:%d/' /home/step/config/ca.json`, s.port)
	run(t, ctx, "docker", "run", "--rm",
		"-v", volumeName+":/home/step",
		"--entrypoint", "/bin/sh",
		"smallstep/step-ca:latest",
		"-c", patchScript,
	)

	// Start the CA server with host networking so it can reach challenge
	// servers on localhost for http-01 validation.
	run(t, ctx, "docker", "run", "-d",
		"--name", s.containerName,
		"--network=host",
		"-v", volumeName+":/home/step",
		"-e", "DOCKER_STEPCA_INIT_PASSWORD=testpassword",
		"smallstep/step-ca:latest",
	)

	// Wait for the server to be ready
	s.waitReady(t, ctx)

	t.Logf("step-ca started on port %d, directory: %s", s.port, s.DirectoryURL())
}

func (s *Server) waitReady(t *testing.T, ctx context.Context) {
	t.Helper()

	client := s.HTTPClient()
	deadline := time.Now().Add(60 * time.Second)

	for time.Now().Before(deadline) {
		resp, err := client.Get(s.DirectoryURL())
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case <-ctx.Done():
			// Grab container logs for debugging
			logs, _ := exec.Command("docker", "logs", s.containerName).CombinedOutput()
			t.Fatalf("step-ca failed to start (context deadline): %s\nlogs: %s", err, string(logs))
		case <-time.After(500 * time.Millisecond):
		}
	}
	logs, _ := exec.Command("docker", "logs", s.containerName).CombinedOutput()
	t.Fatalf("step-ca failed to become ready within 60s\nlogs: %s", string(logs))
}

func (s *Server) stop(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	volumeName := s.containerName + "-vol"

	// Stop and remove container (ignore errors — may already be stopped)
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", s.containerName).Run()
	_ = exec.CommandContext(ctx, "docker", "volume", "rm", "-f", volumeName).Run()
}

// VerifyDirectory fetches the ACME directory and returns the parsed JSON.
// Useful for quick smoke tests.
func (s *Server) VerifyDirectory(t *testing.T) map[string]interface{} {
	t.Helper()

	resp, err := s.HTTPClient().Get(s.DirectoryURL())
	if err != nil {
		t.Fatalf("failed to fetch directory: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read directory body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("directory returned status %d: %s", resp.StatusCode, string(body))
	}

	var dir map[string]interface{}
	if err := json.Unmarshal(body, &dir); err != nil {
		t.Fatalf("failed to parse directory JSON: %v", err)
	}

	// Verify required ACME directory fields
	for _, field := range []string{"newNonce", "newAccount", "newOrder"} {
		if _, ok := dir[field]; !ok {
			t.Errorf("directory missing required field %q", field)
		}
	}

	return dir
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func run(t *testing.T, ctx context.Context, name string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %q failed: %v\noutput: %s", strings.Join(append([]string{name}, args...), " "), err, string(out))
	}
}

func runOutput(t *testing.T, ctx context.Context, name string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %q failed: %v\noutput: %s", strings.Join(append([]string{name}, args...), " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}
