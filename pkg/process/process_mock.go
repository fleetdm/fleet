package process

import (
	"os"

	"github.com/stretchr/testify/mock"
)

type mockOsProcess struct {
	mock.Mock
	OsProcess
}

func (m *mockOsProcess) Signal(sig os.Signal) error {
	args := m.Called(sig)
	err := args.Error(0)
	if err == nil {
		return nil
	}
	return err.(error)
}

func (m *mockOsProcess) Kill() error {
	args := m.Called()
	err := args.Error(0)
	if err == nil {
		return nil
	}
	return err.(error)
}

type mockExecCmd struct {
	mock.Mock
	ExecCmd
}

func (m *mockExecCmd) Start() error {
	args := m.Called()
	err := args.Error(0)
	if err == nil {
		return nil
	}
	return err.(error)
}

func (m *mockExecCmd) Wait() error {
	args := m.Called()
	err := args.Error(0)
	if err == nil {
		return nil
	}
	return err.(error)
}

func (m *mockExecCmd) OsProcess() OsProcess {
	args := m.Called()
	proc := args.Get(0)
	if proc == nil {
		return nil
	}
	return proc.(OsProcess)
}
