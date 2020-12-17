package main

import (
	"context"
	"fmt"
	"os"
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
	err := p.WaitOrKill(context.Background(), os.Interrupt, 1*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-nil OsProcess")
}

func TestWaitOrKillNilSignal(t *testing.T) {
	// Nil signal provided
	mockCmd := &mockExecCmd{}
	mockProcess := &mockOsProcess{}
	defer mock.AssertExpectationsForObjects(t, mockCmd, mockProcess)
	mockCmd.On("OsProcess").Return(mockProcess)

	p := newWithMock(mockCmd)
	err := p.WaitOrKill(context.Background(), nil, 10*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-nil interrupt")
}

func TestWaitOrKillProcessCompleted(t *testing.T) {
	// Process already completed
	mockCmd := &mockExecCmd{}
	mockProcess := &mockOsProcess{}
	defer mock.AssertExpectationsForObjects(t, mockCmd, mockProcess)
	mockCmd.On("OsProcess").Return(mockProcess)
	mockCmd.On("Wait").Return(nil)

	p := newWithMock(mockCmd)
	err := p.WaitOrKill(context.Background(), os.Interrupt, 10*time.Millisecond)
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
	err := p.WaitOrKill(context.Background(), os.Interrupt, 10*time.Millisecond)
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
	mockProcess.On("Signal", os.Interrupt).Return(nil)

	p := newWithMock(mockCmd)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := p.WaitOrKill(ctx, os.Interrupt, 10*time.Millisecond)
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
	mockProcess.On("Signal", os.Interrupt).Return(fmt.Errorf("os: process already finished"))

	p := newWithMock(mockCmd)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := p.WaitOrKill(ctx, os.Interrupt, 5*time.Millisecond)
	require.NoError(t, err)
}

func TestWaitOrKillWaitKilled(t *testing.T) {
	// Process is killed after the wait call and signal
	mockCmd := &mockExecCmd{}
	mockProcess := &mockOsProcess{}
	defer mock.AssertExpectationsForObjects(t, mockCmd, mockProcess)
	mockCmd.On("OsProcess").Return(mockProcess)
	mockCmd.On("Wait").After(10 * time.Millisecond).Return(fmt.Errorf("killed"))
	mockProcess.On("Signal", os.Interrupt).Return(nil)
	mockProcess.On("Kill").Return(nil)

	p := newWithMock(mockCmd)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := p.WaitOrKill(ctx, os.Interrupt, 5*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
