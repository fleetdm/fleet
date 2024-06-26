// Package executablecsrverifier defines the ExecutableCSRVerifier csrverifier.CSRVerifier.
package executablecsrverifier

import (
	"errors"
	"os"
	"os/exec"

	"github.com/go-kit/log"
)

const (
	userExecute os.FileMode = 1 << (6 - 3*iota)
	groupExecute
	otherExecute
)

// New creates a executablecsrverifier.ExecutableCSRVerifier.
func New(path string, logger log.Logger) (*ExecutableCSRVerifier, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	fileMode := fileInfo.Mode()
	if fileMode.IsDir() {
		return nil, errors.New("CSR Verifier executable is a directory")
	}

	filePerm := fileMode.Perm()
	if filePerm&(userExecute|groupExecute|otherExecute) == 0 {
		return nil, errors.New("CSR Verifier executable is not executable")
	}

	return &ExecutableCSRVerifier{executable: path, logger: logger}, nil
}

// ExecutableCSRVerifier implements a csrverifier.CSRVerifier.
// It executes a command, and passes it the raw decrypted CSR.
// If the command exit code is 0, the CSR is considered valid.
// In any other cases, the CSR is considered invalid.
type ExecutableCSRVerifier struct {
	executable string
	logger     log.Logger
}

func (v *ExecutableCSRVerifier) Verify(data []byte) (bool, error) {
	cmd := exec.Command(v.executable) // nolint:gosec

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return false, err
	}

	go func() {
		defer stdin.Close()
		_, _ = stdin.Write(data)
	}()

	err = cmd.Run()
	if err != nil {
		v.logger.Log("err", err)
		// mask the executable error
		return false, nil
	}
	return true, err
}
