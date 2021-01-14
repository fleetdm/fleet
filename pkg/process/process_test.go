package process

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWaitOrKillNilProcess(t *testing.T) {
	// Process not started
	mockCmd := &mockExecCmd{}
	defer mock.AssertExpectationsForObjects(t, mockCmd)
	mockCmd.On("OsProcess").Return(nil)

	p := newWithMock(mockCmd)
	err := p.WaitOrKill(context.Background(), 1*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-nil OsProcess")
}

func TestWaitOrKillProcessCompleted(t *testing.T) {
	// Process already completed
	mockCmd := &mockExecCmd{}
	mockProcess := &mockOsProcess{}
	defer mock.AssertExpectationsForObjects(t, mockCmd, mockProcess)
	mockCmd.On("OsProcess").Return(mockProcess)
	mockCmd.On("Wait").Return(nil)

	p := newWithMock(mockCmd)
	err := p.WaitOrKill(context.Background(), 10*time.Millisecond)
	require.NoError(t, err)
}

func TestWaitOrKillProcessCompletedError(t *testing.T) {
	// Process already completed with error
	mockCmd := &mockExecCmd{}
	mockProcess := &mockOsProcess{}
	defer mock.AssertExpectationsForObjects(t, mockCmd, mockProcess)
	mockCmd.On("OsProcess").Return(mockProcess)
	mockCmd.On("Wait").After(10 * time.Millisecond).Return(fmt.Errorf("super bad"))

	p := newWithMock(mockCmd)
	err := p.WaitOrKill(context.Background(), 10*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "super bad")
}

func TestWaitOrKillWait(t *testing.T) {
	// Process completes after the wait call and after the signal is sent
	mockCmd := &mockExecCmd{}
	mockProcess := &mockOsProcess{}
	defer mock.AssertExpectationsForObjects(t, mockCmd, mockProcess)
	mockCmd.On("OsProcess").Return(mockProcess)
	mockCmd.On("Wait").After(5 * time.Millisecond).Return(nil)
	mockProcess.On("Signal", stopSignal()).Return(nil)

	p := newWithMock(mockCmd)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := p.WaitOrKill(ctx, 10*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestWaitOrKillWaitSignalCompleted(t *testing.T) {
	// Process completes after the wait call and before the signal is sent
	mockCmd := &mockExecCmd{}
	mockProcess := &mockOsProcess{}
	defer mock.AssertExpectationsForObjects(t, mockCmd, mockProcess)
	mockCmd.On("OsProcess").Return(mockProcess)
	mockCmd.On("Wait").After(10 * time.Millisecond).Return(nil)
	mockProcess.On("Signal", stopSignal()).Return(fmt.Errorf("os: process already finished"))

	p := newWithMock(mockCmd)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := p.WaitOrKill(ctx, 5*time.Millisecond)
	require.NoError(t, err)
}

func TestWaitOrKillWaitKilled(t *testing.T) {
	// Process is killed after the wait call and signal
	mockCmd := &mockExecCmd{}
	mockProcess := &mockOsProcess{}
	defer mock.AssertExpectationsForObjects(t, mockCmd, mockProcess)
	mockCmd.On("OsProcess").Return(mockProcess)
	mockCmd.On("Wait").After(10 * time.Millisecond).Return(fmt.Errorf("killed"))
	mockProcess.On("Signal", stopSignal()).Return(nil)
	mockProcess.On("Kill").Return(nil)

	p := newWithMock(mockCmd)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := p.WaitOrKill(ctx, 5*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
