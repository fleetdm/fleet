# 🚀 GitHub Management (GM) Tool

> **Supercharge your GitHub workflow with bulk operations and beautiful terminal UI**

<!-- GIF Demo Space - Add your application demo GIF here -->
*Demo coming soon...*

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

# View estimated tickets
./gm estimated mdm --limit 25
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
