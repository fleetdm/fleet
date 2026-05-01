package main

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// newWorktreeModel is the form a PM fills in to create a new worktree.
// PR 3 keeps it deliberately minimal: a branch name (always created
// from main) and a path that auto-fills based on the branch. Validation
// runs on submit; errors render inline beneath the form.
type newWorktreeModel struct {
	launchRoot string

	branchInput textinput.Model
	pathInput   textinput.Model

	// pathManuallyEdited tracks whether the user has changed the path
	// field. While false, typing in the branch field auto-rewrites the
	// path. Once they touch the path field directly, we stop overwriting
	// their value.
	pathManuallyEdited bool

	// 0 = branch, 1 = path
	focus int

	errMsg string

	// done flips when the form completes successfully — root model reads
	// this to know it can return to the switcher.
	done bool

	// created is set when done is true, so the root model knows what
	// got registered.
	created WorktreeEntry
}

func newNewWorktreeModel(launchRoot string) newWorktreeModel {
	branch := textinput.New()
	branch.Placeholder = "feature/my-thing"
	branch.CharLimit = 200
	branch.Width = 50
	branch.Focus()

	path := textinput.New()
	path.Placeholder = DefaultWorktreePath(launchRoot, "feature-my-thing")
	path.CharLimit = 400
	path.Width = 50

	return newWorktreeModel{
		launchRoot:  launchRoot,
		branchInput: branch,
		pathInput:   path,
		focus:       0,
	}
}

// update handles tea.KeyMsg + the textinput tick messages. Returns the
// next state plus a tea.Cmd if the embedded textinputs need one.
func (n newWorktreeModel) update(msg tea.Msg) (newWorktreeModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "tab", "shift+tab":
			if n.focus == 0 {
				n.branchInput.Blur()
				n.pathInput.Focus()
				n.focus = 1
			} else {
				n.pathInput.Blur()
				n.branchInput.Focus()
				n.focus = 0
			}
			return n, nil
		case "enter":
			if err := validateNewWorktreeForm(
				context.Background(),
				n.launchRoot,
				n.branchInput.Value(),
				n.pathInput.Value(),
			); err != nil {
				n.errMsg = err.Error()
				return n, nil
			}
			// Validation passed — execute the create. This call hits
			// `git worktree add` synchronously; on a clean repo it
			// returns in <1s.
			branch := strings.TrimSpace(n.branchInput.Value())
			path := strings.TrimSpace(n.pathInput.Value())
			if _, err := gitWorktreeAdd(context.Background(), n.launchRoot, branch, path); err != nil {
				n.errMsg = err.Error()
				return n, nil
			}
			n.created = WorktreeEntry{
				Name:   DefaultWorktreeName(path),
				Path:   path,
				Branch: branch,
			}
			n.done = true
			return n, nil
		}
	}

	var cmd tea.Cmd
	switch n.focus {
	case 0:
		oldBranch := n.branchInput.Value()
		n.branchInput, cmd = n.branchInput.Update(msg)
		// Auto-rewrite path field while user hasn't touched it.
		if !n.pathManuallyEdited && n.branchInput.Value() != oldBranch {
			n.pathInput.SetValue(DefaultWorktreePath(n.launchRoot, n.branchInput.Value()))
		}
	case 1:
		oldPath := n.pathInput.Value()
		n.pathInput, cmd = n.pathInput.Update(msg)
		if n.pathInput.Value() != oldPath {
			n.pathManuallyEdited = true
		}
	}
	return n, cmd
}

func (n newWorktreeModel) view(width int) string {
	if width <= 0 {
		width = 80
	}
	header := styleHeaderBrand.Render("Fleet ship") + styleHint.Render("  ·  new worktree")

	branchField := renderField(
		"Branch (created from main)",
		"slashes are allowed; the path field updates automatically",
		n.branchInput.View(),
	)
	pathField := renderField(
		"Path",
		"directory the new worktree lives in",
		n.pathInput.View(),
	)

	body := []string{
		header,
		"",
		branchField,
		"",
		pathField,
	}

	if n.errMsg != "" {
		body = append(body, "", "  "+styleErrorLine(n.errMsg))
	}

	body = append(body,
		"",
		strings.Join([]string{
			styleKey.Render("enter") + " " + styleHint.Render("create"),
			styleKey.Render("tab") + " " + styleHint.Render("next field"),
			styleKey.Render("esc") + " " + styleHint.Render("back"),
		}, styleHint.Render("   ·   ")),
	)

	return stylePane.Width(width - 2).Render(strings.Join(body, "\n"))
}

// styleErrorLine renders a short error message in the error color. Used
// inline in form-style screens.
func styleErrorLine(msg string) string {
	return lipgloss.NewStyle().Foreground(colorErr).Render("✗ " + msg)
}
