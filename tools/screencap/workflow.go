package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// actionKind describes the type of user interaction recorded.
type actionKind string

const (
	actionClick      actionKind = "click"
	actionHover      actionKind = "hover"
	actionRadio      actionKind = "radio"
	actionCheckbox   actionKind = "checkbox"
	actionSelectTab  actionKind = "tab"
	actionToggle     actionKind = "toggle"
)

// recordedAction is a single user interaction captured during recording.
type recordedAction struct {
	Kind     actionKind `json:"kind"`
	Selector string     `json:"selector"`
	Text     string     `json:"text,omitempty"`
}

// workflowStep is a single step in a recorded workflow.
type workflowStep struct {
	Name       string           `json:"name"`
	Path       string           `json:"path"`
	ScrollY    float64          `json:"scroll_y,omitempty"`
	HasModal   bool             `json:"has_modal,omitempty"`
	ModalTitle string           `json:"modal_title,omitempty"`
	Actions    []recordedAction `json:"actions,omitempty"`
}

// workflow is a named, replayable sequence of screenshot steps.
type workflow struct {
	Name  string         `json:"name"`
	Steps []workflowStep `json:"steps"`
}

// workflowsDir returns tools/screencap/workflows/ relative to this source file,
// so workflows live in the repo and can be committed and shared.
func workflowsDir() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot determine source file path")
	}
	dir := filepath.Join(filepath.Dir(thisFile), "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func workflowPath(name string) (string, error) {
	dir, err := workflowsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".json"), nil
}

func saveWorkflow(w *workflow) error {
	path, err := workflowPath(w.Name)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func loadWorkflow(name string) (*workflow, error) {
	path, err := workflowPath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("workflow %q not found: %w", name, err)
	}
	var w workflow
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("invalid workflow file: %w", err)
	}
	return &w, nil
}

func listWorkflows() ([]string, error) {
	dir, err := workflowsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name()[:len(e.Name())-5])
		}
	}
	return names, nil
}

// recorderJS is injected into the page during recording mode.
// It listens for clicks on interactive elements and records them
// with a stable CSS selector so they can be replayed.
const recorderJS = `(() => {
	if (window.__screencapRecorder) return;
	window.__screencapRecorder = true;
	window.__screencapActions = [];

	function getSelector(el) {
		// Try data-text attribute (used by Fleet tabs).
		if (el.dataset && el.dataset.text) {
			return '[data-text="' + el.dataset.text + '"]';
		}
		// Try id.
		if (el.id) {
			return '#' + CSS.escape(el.id);
		}
		// Try unique class combination.
		if (el.className && typeof el.className === 'string') {
			const classes = el.className.trim().split(/\s+/).filter(c => c && !c.startsWith('active') && !c.startsWith('hover'));
			if (classes.length > 0) {
				const sel = '.' + classes.map(c => CSS.escape(c)).join('.');
				if (document.querySelectorAll(sel).length === 1) {
					return sel;
				}
			}
		}
		// Build a path from parent.
		const parts = [];
		let node = el;
		while (node && node !== document.body) {
			let part = node.tagName.toLowerCase();
			if (node.id) {
				parts.unshift('#' + CSS.escape(node.id) + ' > ' + part);
				break;
			}
			const parent = node.parentElement;
			if (parent) {
				const siblings = Array.from(parent.children).filter(c => c.tagName === node.tagName);
				if (siblings.length > 1) {
					part += ':nth-of-type(' + (siblings.indexOf(node) + 1) + ')';
				}
			}
			parts.unshift(part);
			node = parent;
		}
		return parts.join(' > ');
	}

	function classify(el) {
		const tag = el.tagName.toLowerCase();
		const type = (el.getAttribute('type') || '').toLowerCase();
		if (tag === 'input' && type === 'radio') return 'radio';
		if (tag === 'input' && type === 'checkbox') return 'checkbox';
		if (el.getAttribute('role') === 'tab' || el.dataset.text) return 'tab';
		if (el.classList.contains('toggle') || el.getAttribute('role') === 'switch') return 'toggle';
		return 'click';
	}

	document.addEventListener('click', (e) => {
		// Walk up to find the meaningful interactive element.
		let target = e.target;
		for (let i = 0; i < 5 && target; i++) {
			const tag = target.tagName.toLowerCase();
			if (tag === 'button' || tag === 'a' || tag === 'input' || tag === 'select' ||
				target.getAttribute('role') === 'tab' || target.getAttribute('role') === 'switch' ||
				target.dataset.text || target.classList.contains('toggle')) {
				break;
			}
			target = target.parentElement;
		}
		if (!target) target = e.target;

		const action = {
			kind: classify(target),
			selector: getSelector(target),
			text: (target.textContent || '').trim().substring(0, 100),
		};
		window.__screencapActions.push(action);
	}, true);

	// Track hovers on elements with tooltips.
	document.addEventListener('mouseover', (e) => {
		const target = e.target;
		if (target.title || target.getAttribute('data-tooltip') || target.getAttribute('aria-describedby') ||
			target.closest('[data-tip]') || target.closest('[data-tooltip-id]')) {
			const el = target.closest('[data-tip]') || target.closest('[data-tooltip-id]') || target;
			const action = {
				kind: 'hover',
				selector: getSelector(el),
				text: (el.title || el.getAttribute('data-tip') || el.getAttribute('data-tooltip') || '').substring(0, 100),
			};
			// Deduplicate consecutive hovers on the same element.
			const last = window.__screencapActions[window.__screencapActions.length - 1];
			if (!last || last.kind !== 'hover' || last.selector !== action.selector) {
				window.__screencapActions.push(action);
			}
		}
	}, true);
})()`

// drainActionsJS returns the recorded actions and clears the buffer.
const drainActionsJS = `(() => {
	const actions = window.__screencapActions || [];
	window.__screencapActions = [];
	return actions;
})()`
