# 🚀 GitHub Management (GM) Tool

> **Supercharge your GitHub workflow with bulk operations and beautiful terminal UI**

<!-- GIF Demo Space - Add your application demo GIF here -->
![GM Tool Demo](assets/gm-demo-labels.gif)

![GM Kickoff / filter Demo](assets/gm-demo-kickoff-filter.gif)

---

## ✨ What is GM?

GM (GitHub Management) is a powerful command-line tool that brings **bulk operations** and **beautiful visualization** to GitHub issue management. Built with ❤️ using [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Glamour](https://github.com/charmbracelet/glamour), it transforms tedious GitHub workflows into delightful interactive experiences.

## 🎯 Features

### 🔍 **Smart Issue Discovery**
- **Search Issues**: Powerful GitHub search syntax support
- **Project Views**: Browse issues by project with estimates
- **Scrollable Lists**: Navigate through hundreds of issues with ease
- **Live Filtering**: Press `/` to filter issues by number, title, labels, or description
- **Real-time Filtering**: Filter updates instantly as you type

### 📋 **Detailed Issue Views**
- **Full Issue Details**: Press `o` to view complete issue information
- **Markdown Rendering**: Beautiful, styled markdown with syntax highlighting
- **Scrollable Content**: Navigate through long descriptions smoothly
- **Metadata Display**: Labels, estimates, assignees, milestones at a glance

### ⚡ **Bulk Operations & Workflows**
- **🏷️ Bulk Label Management**: Add/remove labels across multiple issues
- **🚀 Sprint Kickoff**: Move issues from drafting to active sprint
- **📊 Milestone Close**: Batch close milestones and move issues
- **↩️ Kick Out of Sprint**: Remove issues from current sprint back to drafting
- **📈 Progress Tracking**: Real-time visual progress with async operations

### 🛠️ **Developer Experience**
- **File-based Logging**: All operations logged to `dgm.log` for debugging
- **Command Debugging**: Track all commands and arguments
- **Error Handling**: Graceful error recovery with detailed messages
- **GitHub CLI Integration**: Leverages your existing GitHub authentication

## 🚀 Quick Start

### Prerequisites
- [GitHub CLI](https://cli.github.com/) installed and authenticated
```
gh auth login
gh auth refresh -s project # This will give gh project access which is not included by default
```
- Go 1.24+ for building from source

### Installation

```bash
# Clone the repository
git clone https://github.com/fleetdm/fleet.git
cd fleet/tools/github-manage

# Build the tool
go build -o gm cmd/gm/*.go
# or
make

# Make it executable (optional - add to PATH)
chmod +x gm
```

### Basic Usage

```bash
# Search for issues
./gm issues --search "is:open label:bug"

# View project items
./gm project 58 --limit 50
# Don't know your project number off the top of your head?
# There are some easy to use aliases defined in pkg/ghapi/project.go in `Aliases`

# Pre-sprint report for one or more teams
# Single team (alias or project id)
./gm pre-sprint report mdm
# Multiple teams (comma-separated)
./gm pre-sprint report mdm,soft --limit 1000

# CSV format for spreadsheet use (outputs values per team in provided order)
./gm pre-sprint report mdm,soft --format csv

# Priority (P0/P1) issue report over time
# Buckets issues by the week they were created and breaks them down by
# product group ("#g-" label) per month so you can spot ownership trends.
./gm reports priority
# Override the window (default is the last 6 months)
./gm reports priority --months 12
# Report on a single priority, or add P2
./gm reports priority --priority P0
./gm reports priority --priority P0,P1,P2
# Machine-readable output
./gm reports priority --format json
```

## 🎮 Interactive Controls

### 📝 **Issue List Navigation**
| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Move cursor up/down |
| `PgUp/PgDn` or `Ctrl+b/f` | Page up/down |
| `Home/End` or `Ctrl+a/e` | Jump to first/last issue |
| `Space/Enter/x` | Toggle issue selection |
| `/` | **Start filtering issues** |
| `o` | **View full issue details** |
| `w` | Open workflow menu |
| `q` | Quit application |

### 🔍 **Filter Mode**
| Key | Action |
|-----|--------|
| `Type` | Filter by number, title, labels, description |
| `Backspace` | Remove last character from filter |
| `Enter` | **Apply filter and return to list** |
| `Esc` | **Clear filter and return to list** |
| `q` | Quit application |

### 📖 **Issue Detail View**
| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Scroll up/down |
| `PgUp/PgDn` | Page up/down |
| `Home/End` | Jump to top/bottom |
| `Esc` | **Return to issue list** |
| `q` | Quit application |

### ⚡ **Workflow Operations**
1. **Filter Issues**: Press `/` to narrow down the list by typing keywords
2. **Select Issues**: Use `Space/Enter` to select multiple issues (selections persist across filters)
3. **Start Workflow**: Press `w` to open workflow menu
4. **Choose Operation**: Navigate with `↑/↓`, confirm with `Enter`
5. **Watch Progress**: Real-time progress bars and status updates
6. **Review Results**: Success/failure summary with error details

## 🔧 Advanced Features

### 📊 **Project Management**
- **Project Integration**: Seamlessly work with GitHub Projects
- **Estimate Tracking**: View and sync story point estimates
- **Sprint Management**: Automate sprint transitions
- **Status Updates**: Bulk status changes across project items

### 🎨 **Beautiful UI**
- **Syntax Highlighting**: Code blocks in issue descriptions rendered beautifully
- **Progress Visualization**: Animated progress bars for long operations
- **Color-coded Status**: Visual indicators for task states
- **Responsive Design**: Adapts to your terminal size

### 🔍 **Logging & Debugging**
- **Comprehensive Logging**: All operations logged to `dgm.log`
- **Command Tracing**: Debug mode tracks all GitHub CLI commands
- **Error Context**: Detailed error messages with actionable information
- **Performance Metrics**: Operation timing and success rates

## 🎩 jarvis — Personal Work Dashboard

`./gm jarvis` opens an interactive TUI that gathers all of your open GitHub work
onto one screen, ordered by leverage. It pulls three sources — pull requests you
authored, pull requests awaiting your review, and issues assigned to you (with
their project board Status) — and refreshes on demand (`r` for the highlighted
item, `R` for everything). Fetches are cached for 4h so it opens instantly.

First run walks you through a role picker and a project-board picker, saved to
`~/.config/gm/jarvis/config.json`. jarvis needs the gh `project` scope; if the
board listing fails it tells you to run `gh auth refresh -s project`.

### How items land in each section

Every item is placed in the **highest** section that applies, so the most
leveraged work is always at the top:

| Section | What ends up here |
|---------|-------------------|
| **PROJECT VIEW** | Issues assigned to you on your primary project boards, plus a count of the Ready backlog. Grouped by board; each board always shows (even when empty) so you can pick up new work. |
| **WAITING ON YOU** | Others are blocked on you — changes/comments bounced back on your PR, unresolved review threads, or a PR you reviewed that changed since. |
| **QUICK WINS** | Your PRs that can merge right now: CI green, approved, no conflicts. |
| **NEEDS YOUR HANDS** | Your own work needing action: merge conflicts, failing CI, or an assigned issue with no PR yet. |
| **CLAUDE SESSIONS** | Local Claude sessions waiting on your reply. |
| **REVIEW QUEUE** | PRs awaiting a first review from you. |
| **COLD** | Waiting on others or gone stale: CI still running, awaiting others' review, drafts, stale assignments. |

### Triage — and automatic reappearance

- `x` mark done · `d` dismiss · `s` snooze (1h / 4h / tomorrow / 1 week) · `u` clear · `H` show hidden
- **Anything you hide comes back on its own when it gets new activity.** When you
  mark something done (or dismiss/snooze it), jarvis records the item's
  last-updated time. If the issue or PR is updated after that — a new comment, a
  push, a status change — it automatically returns to its section on the next
  refresh. Nothing stays buried once someone touches it again.
- Merged/closed PRs and closed issues are marked done for you and drop off.

### Start work

Press `w` on an issue to start work: name a branch, then pick a local clone to
work in — or choose **＋ create new fleet-… working dir** to clone a fresh copy
first (you name it, jarvis prefixes `fleet-` and clones under your first
`clone_base_dirs` entry). Either way jarvis branches off main, sets the issue to
In progress, and launches a Claude session seeded with the issue context.

### 🌿 Branch cleanup

Press `B` for the branch-cleanup view. It scans every git repo matching
`branch_scan_glob` (default `fleet*`) under your `clone_base_dirs` and lists each
local branch tagged by state:

- `pushed` — fully on origin, safe to delete (recoverable with a fetch)
- `ahead` — has local commits not yet pushed
- `gone` — the upstream was deleted (e.g. the PR merged from the web); shows after a prune
- `local` — never pushed

Actions: `d` delete the selected branch, `p` delete all `pushed` branches, `D`
delete everything except main/master. Press `F` first to `git fetch --prune`
every repo so branches whose merged remote was deleted surface as `gone`. The
checked-out branch and main/master are always protected, and every delete asks
for confirmation.

## 🏗️ Architecture

GM is built with modern Go patterns and best practices:

- **🧩 Modular Design**: Separate packages for GitHub API, logging, and UI
- **⚡ Async Operations**: Non-blocking bulk operations with real-time updates
- **🔄 State Management**: Robust state handling with Bubble Tea
- **🎭 Clean UI**: Separation of business logic and presentation
- **📝 Comprehensive Logging**: Debug-friendly logging throughout

## 🤝 Contributing

We welcome contributions! Whether it's:
- 🐛 Bug fixes
- ✨ New features
- 📚 Documentation improvements
- 🎨 UI enhancements

## 📄 License

This project is part of the [Fleet](https://github.com/fleetdm/fleet) repository and follows the same licensing terms.

---

<div align="center">

**Built with 💪 by the Fleet team**

*Making GitHub management a joy, one bulk operation at a time*

</div>
