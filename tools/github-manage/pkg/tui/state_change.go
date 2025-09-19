package tui

import (
	"fmt"
	"time"

	"fleetdm/gm/pkg/ghapi"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) HandleStateChange(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case issuesLoadedMsg:
		m.choices = msg.issues
		m.totalCount = len(m.choices)
		m.totalAvailable = msg.totalAvailable
		m.rawFetchedCount = msg.rawFetched
		m.workflowState = NormalMode
		m.applyFilter()         // Initialize filter state
		m.adjustViewForCursor() // Ensure view is properly initialized
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	case progress.FrameMsg:
		progressModel, cmd := m.overallProgress.Update(msg)
		m.overallProgress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
	case tea.WindowSizeMsg:
		// Update viewport size when window is resized
		m.detailViewport.Width = msg.Width - 4
		m.detailViewport.Height = msg.Height - 6
	case processTaskMsg:
		// This case is no longer used since we've moved to AsyncManager
		// Ignore these messages
		return m, nil
	case startAsyncWorkflowMsg:
		if m.workflowState == WorkflowRunning {
			// Store the actions and start AsyncManager
			m.currentActions = msg.actions

			// Create status channel
			m.statusChan = make(chan ghapi.Status)

			// Start AsyncManager in a goroutine
			go ghapi.AsyncManager(msg.actions, m.statusChan)

			// Return command to listen for status updates
			return m, m.listenForAsyncStatus()
		}
	case actionStatusMsg:
		if m.workflowState == WorkflowRunning && msg.index < len(m.tasks) {
			// Update the task based on the action status
			if msg.state == "success" {
				m.tasks[msg.index].Status = TaskSuccess
				m.tasks[msg.index].Progress = 1.0
			} else if msg.state == "error" {
				m.tasks[msg.index].Status = TaskError
				m.tasks[msg.index].Progress = 0.0
			}
			m.currentTask = msg.index

			// Update overall progress
			completedTasks := 0
			for _, task := range m.tasks {
				if task.Status == TaskSuccess {
					completedTasks++
				}
			}
			overallPercent := float64(completedTasks) / float64(len(m.tasks))
			cmds = append(cmds, m.overallProgress.SetPercent(overallPercent))

			// Check if we have an error
			if msg.state == "error" {
				m.workflowState = WorkflowComplete
				m.errorMessage = fmt.Sprintf("Task %d failed", msg.index)
			} else if completedTasks == len(m.tasks) {
				// All tasks completed successfully
				m.workflowState = WorkflowComplete
			}
			// Note: Sequential processing is now handled by AsyncManager,
			// so we don't trigger next action processing here
		}
	case asyncStatusMsg:
		if msg.status.State == "done" {
			// AsyncManager is finished, all actions completed
			m.workflowState = WorkflowComplete
			m.statusChan = nil
			return m, nil
		}

		if m.workflowState == WorkflowRunning && msg.status.Index < len(m.tasks) {
			// Update the task based on the async status
			if msg.status.State == "success" {
				m.tasks[msg.status.Index].Status = TaskSuccess
				m.tasks[msg.status.Index].Progress = 1.0
			} else if msg.status.State == "error" {
				m.tasks[msg.status.Index].Status = TaskError
				m.tasks[msg.status.Index].Progress = 0.0
			}
			m.currentTask = msg.status.Index + 1

			// Update overall progress
			completedTasks := 0
			for _, task := range m.tasks {
				if task.Status == TaskSuccess {
					completedTasks++
				}
			}
			overallPercent := float64(completedTasks) / float64(len(m.tasks))
			cmds = append(cmds, m.overallProgress.SetPercent(overallPercent))

			// Check if we have an error
			if msg.status.State == "error" {
				m.workflowState = WorkflowComplete
				m.errorMessage = fmt.Sprintf("Task %d failed", msg.status.Index)
				// Channel will be closed by AsyncManager
				m.statusChan = nil
			} else if completedTasks == len(m.tasks) {
				// All tasks completed successfully
				m.workflowState = WorkflowComplete
				// Channel will be closed by AsyncManager
				m.statusChan = nil
			} else {
				// Continue listening for more status updates
				cmds = append(cmds, m.listenForAsyncStatus())
			}
		}
	case taskUpdateMsg:
		if msg.taskID < len(m.tasks) {
			m.tasks[msg.taskID].Progress = msg.progress
			m.tasks[msg.taskID].Status = msg.status
			if msg.err != nil {
				m.tasks[msg.taskID].Error = msg.err
			}
			m.currentTask = msg.taskID

			// If this is an error, mark workflow as complete
			if msg.status == TaskError {
				m.workflowState = WorkflowComplete
				m.errorMessage = fmt.Sprintf("Task failed: %v", msg.err)
				cmds = append(cmds, m.overallProgress.SetPercent(1.0))
			} else {
				// Update overall progress
				totalProgress := float64(0)
				completedTasks := 0
				for _, task := range m.tasks {
					totalProgress += task.Progress
					if task.Status == TaskSuccess {
						completedTasks++
					}
				}
				overallPercent := totalProgress / float64(len(m.tasks))
				cmds = append(cmds, m.overallProgress.SetPercent(overallPercent))

				// Check if all tasks are completed successfully
				if completedTasks == len(m.tasks) {
					m.workflowState = WorkflowComplete
					// Will show success summary in the view
				} else if msg.status == TaskSuccess && (m.workflowType == BulkAddLabel || m.workflowType == BulkRemoveLabel) {
					// For bulk label operations, trigger the next task with a delay to ensure serialization
					nextTaskID := msg.taskID + 1
					if nextTaskID < len(m.tasks) {
						cmds = append(cmds, func() tea.Msg {
							// Add a small delay before processing the next task
							time.Sleep(50 * time.Millisecond)
							return processTaskMsg{taskID: nextTaskID}
						})
					}
				}
			}
		}
	case workflowExitMsg:
		m.exitMessage = msg.message
		return m, tea.Quit
	case workflowCompleteMsg:
		m.workflowState = WorkflowComplete
		if !msg.success {
			m.errorMessage = msg.message
		}
		// Set overall progress to 100%
		cmds = append(cmds, m.overallProgress.SetPercent(1.0))
	case workflowResultMsg:
		// Legacy handler - convert to new system
		if msg.err != nil {
			m.workflowState = WorkflowComplete
			m.errorMessage = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.workflowState = WorkflowComplete
		}
		cmds = append(cmds, m.overallProgress.SetPercent(1.0))
	}

	return m, tea.Batch(cmds...)
}
