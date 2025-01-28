package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/fleetdm/fleet/v4/server/service"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Slots correspond to docker-compose fleet services, either fleet-a or fleet-b
const (
	slotA = "a"
	slotB = "b"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Fleet represents the fleet server and its dependencies used for testing.
type Fleet struct {
	// ProjectName is the docker compose project name
	ProjectName string
	// FilePath is the path to the docker-compose.yml
	FilePath string
	// Version is the active fleet version.
	Version string
	// Token is the fleet token used for authentication
	Token string

	dockerClient client.ContainerAPIClient
}

// NewFleet starts fleet and it's dependencies with the specified version.
func NewFleet(t *testing.T, version string) *Fleet {
	// don't use test name because it will be normalized
	//nolint:gosec // does not need to be secure for tests
	projectName := "fleet-test-" + strconv.FormatUint(rand.Uint64(), 16)

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("create docker client: %v", err)
	}

	f := &Fleet{
		ProjectName:  projectName,
		FilePath:     "docker-compose.yaml",
		Version:      version,
		dockerClient: dockerClient,
	}

	t.Cleanup(f.cleanup)

	if err := f.Start(); err != nil {
		t.Fatalf("start fleet: %v", err)
	}

	return f
}

func (f *Fleet) Start() error {
	env := map[string]string{
		"FLEET_VERSION_A": f.Version,
	}
	_, err := f.execCompose(env, "pull", "--parallel")
	if err != nil {
		return err
	}

	// start mysql and wait until ready
	_, err = f.execCompose(env, "up", "-d", "mysql")
	if err != nil {
		return err
	}
	if err := f.waitMYSQL(); err != nil {
		return err
	}

	// run the migrations using the fleet-a service
	_, err = f.execCompose(env, "run", "-T", "fleet-a", "fleet", "prepare", "db", "--no-prompt")
	if err != nil {
		return err
	}

	// start fleet-a
	_, err = f.execCompose(env, "up", "-d", "fleet-a", "fleet")
	if err != nil {
		return err
	}

	// copy the nginx conf and reload nginx without creating a new container
	srcPath := filepath.Join("nginx", "fleet-a.conf")
	_, err = f.execCompose(env, "cp", srcPath, "fleet:/etc/nginx/conf.d/default.conf")
	if err != nil {
		return err
	}

	_, err = f.execCompose(env, "exec", "-T", "fleet", "nginx", "-s", "reload")
	if err != nil {
		return err
	}

	if err := f.waitFleet(slotA); err != nil {
		return err
	}

	if err := f.setupFleet(); err != nil {
		return err
	}

	return nil
}

// Client returns a fleet client that uses the fleet API.
func (f *Fleet) Client() (*service.Client, error) {
	port, err := f.getPublicPort("fleet", 443)
	if err != nil {
		return nil, fmt.Errorf("get fleet port: %v", err)
	}

	address := fmt.Sprintf("https://localhost:%d", port)
	client, err := service.NewClient(address, true, "", "")
	if err != nil {
		return nil, err
	}

	client.SetToken(f.Token)

	return client, nil
}

func (f *Fleet) setupFleet() error {
	client, err := f.Client()
	if err != nil {
		return err
	}

	token, err := client.Setup("admin@example.com", "Admin", "password123#", "Fleet Test")
	if err != nil {
		return err
	}
	f.Token = token

	return nil
}

func (f *Fleet) waitMYSQL() error {
	// get the random mysql host port assigned by docker
	port, err := f.getPublicPort("mysql", 3306)
	if err != nil {
		return err
	}

	dsn := fmt.Sprintf("fleet:fleet@tcp(localhost:%d)/fleet", port)

	retryInterval := 5 * time.Second
	timeout := 1 * time.Minute

	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)
	for {
		select {
		case <-timeoutChan:
			return fmt.Errorf("db connection failed after %s", timeout)
		case <-ticker.C:
			db, err := sqlx.Connect("mysql", dsn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to connect to db: %v\n", err)
			} else {
				db.Close()
				return nil
			}
		}
	}
}

func (f *Fleet) getPublicPort(serviceName string, privatePort uint16) (uint16, error) {
	containerName := fmt.Sprintf("%s-%s-1", f.ProjectName, serviceName)

	// get the random fleet host port assigned by docker
	argsName := filters.Arg("name", containerName)
	containers, err := f.dockerClient.ContainerList(context.TODO(), container.ListOptions{Filters: filters.NewArgs(argsName), All: true})
	if err != nil {
		return 0, err
	}
	if len(containers) == 0 {
		return 0, errors.New("no containers found")
	}
	for _, port := range containers[0].Ports {
		if port.PrivatePort == privatePort {
			return port.PublicPort, nil
		}
	}
	return 0, errors.New("private port not found")
}

func (f *Fleet) waitFleet(slot string) error {
	containerName := fmt.Sprintf("%s-fleet-%s-1", f.ProjectName, slot)

	// get the random fleet host port assigned by docker
	argsName := filters.Arg("name", containerName)
	containers, err := f.dockerClient.ContainerList(context.TODO(), container.ListOptions{Filters: filters.NewArgs(argsName), All: true})
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		return errors.New("no fleet container found")
	}
	port := containers[0].Ports[0].PublicPort
	healthURL := fmt.Sprintf("http://localhost:%d/healthz", port)

	retryStrategy := backoff.NewExponentialBackOff()
	retryStrategy.MaxInterval = 1 * time.Second

	if err := backoff.Retry(
		func() error {
			//nolint:gosec // G107: Ok to trust docker here
			resp, err := http.Get(healthURL)
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("non-200 status code: %d", resp.StatusCode)
			}
			return nil
		},
		retryStrategy,
	); err != nil {
		return fmt.Errorf("check health: %v", err)
	}

	return nil
}

func (f *Fleet) cleanup() {
	output, err := f.execCompose(nil, "down", "-v", "--remove-orphans")
	if err != nil {
		fmt.Fprintf(os.Stderr, "stop fleet: %v %s", err, output)
	}
}

func (f *Fleet) execCompose(env map[string]string, args ...string) (string, error) {
	// docker compose variables via environment eg FLEET_VERSION_A
	e := os.Environ()
	for k, v := range env {
		e = append(e, fmt.Sprintf("%s=%s", k, v))
	}

	// prepend default args
	args = append([]string{
		"compose",
		"--project-name", f.ProjectName,
		"--file", f.FilePath,
	}, args...)

	var stdout, stderr bytes.Buffer

	cmd := exec.Command("docker", args...)
	cmd.Env = e
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("docker: %v %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// StartHost starts an osquery host using docker-compose and enrolls it with fleet.
// Returns the container ID which is also the hostname and osquery host ID.
func (f *Fleet) StartHost() (string, error) {
	// get the enroll secret
	client, err := f.Client()
	if err != nil {
		return "", err
	}

	enrollSecretSpec, err := client.GetEnrollSecretSpec()
	if err != nil {
		return "", err
	}
	if len(enrollSecretSpec.Secrets) == 0 {
		return "", errors.New("no enroll secret found")
	}

	enrollSecret := enrollSecretSpec.Secrets[0].Secret

	env := map[string]string{
		"ENROLL_SECRET": enrollSecret,
	}
	output, err := f.execCompose(env, "run", "-d", "-T", "osquery")
	if err != nil {
		return "", err
	}

	// get the container id
	containerID := output[:len(output)-1] // strip the newline from output

	// inspect the container to get the hostname
	containerJSON, err := f.dockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return "", fmt.Errorf("inspect container: %v", err)
	}
	hostname := containerJSON.Config.Hostname

	return hostname, nil
}

// Upgrade upgrades fleet to a specified version.
func (f *Fleet) Upgrade(toVersion string) error {
	env := map[string]string{
		"FLEET_VERSION_B": toVersion,
	}

	// run migrations using fleet-b
	serviceName := "fleet-b"
	_, err := f.execCompose(env, "run", "-T", serviceName, "fleet", "prepare", "db", "--no-prompt")
	if err != nil {
		return fmt.Errorf("run migrations: %v", err)
	}

	// start the service
	_, err = f.execCompose(env, "up", "-d", serviceName)
	if err != nil {
		return fmt.Errorf("start fleet: %v", err)
	}

	// wait until healthy
	if err := f.waitFleet(slotB); err != nil {
		return fmt.Errorf("wait for fleet to be healthy: %v", err)
	}

	// copy the nginx conf and reload nginx without creating a new container
	srcPath := filepath.Join("nginx", "fleet-b.conf")
	_, err = f.execCompose(env, "cp", srcPath, "fleet:/etc/nginx/conf.d/default.conf")
	if err != nil {
		return err
	}

	_, err = f.execCompose(env, "exec", "-T", "fleet", "nginx", "-s", "reload")
	if err != nil {
		return err
	}

	f.Version = toVersion

	return nil
}
