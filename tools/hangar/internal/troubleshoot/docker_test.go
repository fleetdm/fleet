package troubleshoot

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseContainers(t *testing.T) {
	raw := strings.Join([]string{
		"3f2a9c1b7d4e\tfleet-mysql-1\tmysql:8.0.36\tfleet\tmysql\tUp 2 hours\t0.0.0.0:3306->3306/tcp, [::]:3306->3306/tcp, 33060/tcp",
		"9b8c7d6e5f4a\tfleet-s2-minio-1\tminio/minio\tfleet-s2\tminio\tUp 5 minutes\t0.0.0.0:9010-9011->9000-9001/tcp",
		"0a1b2c3d4e5f\tgifted_wu\tfleetdm/wix:latest\t\t\tUp 30 seconds\t",
		"not\tenough\tfields",
		"",
	}, "\n")
	got := parseContainers(raw)
	want := []Container{
		{ID: "3f2a9c1b7d4e", Name: "fleet-mysql-1", Image: "mysql:8.0.36", Project: "fleet", Service: "mysql", Status: "Up 2 hours", Ports: []uint16{3306}},
		{ID: "9b8c7d6e5f4a", Name: "fleet-s2-minio-1", Image: "minio/minio", Project: "fleet-s2", Service: "minio", Status: "Up 5 minutes", Ports: []uint16{9010, 9011}},
		{ID: "0a1b2c3d4e5f", Name: "gifted_wu", Image: "fleetdm/wix:latest", Status: "Up 30 seconds", Ports: []uint16{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseContainers =\n%+v\nwant\n%+v", got, want)
	}
	if got := parseContainers(""); got == nil || len(got) != 0 {
		t.Errorf("no containers = %#v, want an empty (non-nil) list", got)
	}
}

func TestPublishedPorts(t *testing.T) {
	for in, want := range map[string][]uint16{
		"":                             {},
		"6379/tcp":                     {},
		"127.0.0.1:16379->6379/tcp":    {16379},
		"0.0.0.0:1-65535->1-65535/tcp": {1}, // absurd range: first port only
	} {
		if got := publishedPorts(in); !reflect.DeepEqual(got, want) {
			t.Errorf("publishedPorts(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestStderrError(t *testing.T) {
	for in, want := range map[string]string{
		"Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?":                                                                                                                           "docker isn't running; start Docker Desktop",
		"failed to connect to the docker API at unix:///Users/me/.docker/run/docker.sock; check if the path is correct and if the daemon is running: dial unix /Users/me/.docker/run/docker.sock: connect: no such file or directory": "docker isn't running; start Docker Desktop",
		"Error response from daemon: No such container: 3f2a9c1b7d4e":                                                                                                                                                                 "Error response from daemon: No such container: 3f2a9c1b7d4e",
	} {
		if got := stderrError(in).Error(); got != want {
			t.Errorf("stderrError(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStopContainersRejectsNonIDs(t *testing.T) {
	for _, id := range []string{"-f", "--time=0", "fleet-mysql-1", "3F2A9C1B7D4E", ""} {
		if err := StopContainers([]string{id}); err == nil || !strings.Contains(err.Error(), "not a container id") {
			t.Errorf("StopContainers(%q) = %v, want a not-a-container-id error", id, err)
		}
	}
	if err := StopContainers(nil); err != nil {
		t.Errorf("StopContainers(nil) = %v", err)
	}
}
