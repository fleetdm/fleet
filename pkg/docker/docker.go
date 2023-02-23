package docker

import (
	"errors"
	"os/exec"
)

const (
	dockerComposeV1 int = iota
	dockerComposeV2
)

type Compose struct {
	version int
}

func (d *Compose) String() string {
	if d.version == dockerComposeV1 {
		return "`docker-compose`"
	}

	return "`docker compose`"
}

func (d *Compose) Command(arg ...string) *exec.Cmd {
	if d.version == dockerComposeV1 {
		return exec.Command("docker-compose", arg...)
	}

	return exec.Command("docker", append([]string{"compose"}, arg...)...)
}

func NewCompose() (*Compose, error) {
	// first, check if `docker compose` is available
	if err := exec.Command("docker", "compose").Run(); err == nil {
		return &Compose{dockerComposeV2}, nil
	}

	// if not, try to use `docker-compose`
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return &Compose{dockerComposeV1}, nil
	}

	return nil, errors.New("docker compose not found")
}
