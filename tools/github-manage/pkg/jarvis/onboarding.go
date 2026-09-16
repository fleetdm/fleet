package jarvis

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"fleetdm/gm/pkg/ghapi"
)

// onboardVisibleRows is how many project rows the picker shows at once; the list
// scrolls to keep the cursor in view when there are more.
const onboardVisibleRows = 12

// fuzzyScore reports whether query matches target as a case-insensitive
// subsequence, and a score that rewards matches that are early and consecutive so
// the best matches sort first. An empty query matches everything with score 0.
func fuzzyScore(query, target string) (int, bool) {
	if query == "" {
		return 0, true
	}
	q := strings.ToLower(query)
	t := strings.ToLower(target)
	score, ti, consecutive := 0, 0, 0
	for qi := 0; qi < len(q); qi++ {
		c := q[qi]
		matched := false
		for ; ti < len(t); ti++ {
			if t[ti] == c {
				score += 1 + consecutive // consecutive runs are worth more
				consecutive++
				ti++
				matched = true
				break
			}
			consecutive = 0
		}
		if !matched {
			return 0, false
		}
	}
	return score, true
}

// onboardModel is the first-run picker: a fuzzy-searchable, multi-select list of
// the org's project boards, used to seed primary_projects in a fresh config.
type onboardModel struct {
	owner    string
	projects []ghapi.OrgProject // open projects, source order
	filter   textinput.Model
	matches  []int            // indices into projects, filtered + ranked
	selected map[int]struct{} // selected project indices
	cursor   int              // index into matches
	top      int              // first visible row in matches (scroll offset)

	confirmed bool // enter pressed
	cancelled bool // esc / ctrl+c
	embedded  bool // running inside the dashboard (via P) rather than as first-run setup
	width     int
}

func newOnboardModel(owner string, projects []ghapi.OrgProject) *onboardModel {
	ti := textinput.New()
	ti.Placeholder = "type to fuzzy-search projects…"
	ti.Focus()
	ti.CharLimit = 80
	m := &onboardModel{
		owner:    owner,
		projects: projects,
		filter:   ti,
		selected: map[int]struct{}{},
		width:    100,
	}
	m.recompute()
	return m
}

func (m *onboardModel) Init() tea.Cmd { return textinput.Blink }

// seedSelection pre-checks the projects that current config entries resolve to, so
// re-opening the picker shows the existing selection. Matching mirrors
// resolveProject: numeric ID / gm alias first, then a case-insensitive name match.
func (m *onboardModel) seedSelection(current []string) {
	for _, entry := range current {
		if id, err := ghapi.ResolveProjectID(entry); err == nil {
			for i, p := range m.projects {
				if p.Number == id {
					m.selected[i] = struct{}{}
				}
			}
			continue
		}
		want := normalizeProjectName(entry)
		for i, p := range m.projects {
			if strings.Contains(strings.ToLower(p.Title), want) {
				m.selected[i] = struct{}{}
			}
		}
	}
}

func (m *onboardModel) recompute() {
	type scored struct {
		idx, score int
	}
	q := strings.TrimSpace(m.filter.Value())
	var hits []scored
	for i, p := range m.projects {
		// Match against the clean handle and the raw title so both "apple" and the
		// emoji-prefixed title work.
		s1, ok1 := fuzzyScore(q, projectHandle(p.Title))
		s2, ok2 := fuzzyScore(q, p.Title)
		if !ok1 && !ok2 {
			continue
		}
		score := s1
		if s2 > score {
			score = s2
		}
		hits = append(hits, scored{i, score})
	}
	sort.SliceStable(hits, func(a, b int) bool {
		if hits[a].score != hits[b].score {
			return hits[a].score > hits[b].score
		}
		return m.projects[hits[a].idx].Number < m.projects[hits[b].idx].Number
	})
	m.matches = m.matches[:0]
	for _, h := range hits {
		m.matches = append(m.matches, h.idx)
	}
	if m.cursor >= len(m.matches) {
		m.cursor = len(m.matches) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.clampScroll()
}

func (m *onboardModel) clampScroll() {
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+onboardVisibleRows {
		m.top = m.cursor - onboardVisibleRows + 1
	}
	if m.top < 0 {
		m.top = 0
	}
}

func (m *onboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Quit the whole app, standalone or embedded.
			m.cancelled = true
			return m, tea.Quit
		case "esc":
			m.cancelled = true
			if m.embedded {
				return m, nil // hand control back to the dashboard
			}
			return m, tea.Quit
		case "enter":
			m.confirmed = true
			if m.embedded {
				return m, nil
			}
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
				m.clampScroll()
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.matches)-1 {
				m.cursor++
				m.clampScroll()
			}
			return m, nil
		case "tab", " ":
			// Toggle selection of the highlighted project. (Space also works even
			// though the filter has focus — projects have no spaces to type.)
			if m.cursor >= 0 && m.cursor < len(m.matches) {
				idx := m.matches[m.cursor]
				if _, ok := m.selected[idx]; ok {
					delete(m.selected, idx)
				} else {
					m.selected[idx] = struct{}{}
				}
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	prev := m.filter.Value()
	m.filter, cmd = m.filter.Update(msg)
	if m.filter.Value() != prev {
		m.recompute()
	}
	return m, cmd
}

func (m *onboardModel) View() string {
	var b strings.Builder
	if m.embedded {
		b.WriteString(headerStyle.Render(" Select projects ") + "\n\n")
		b.WriteString("Pick the project board(s) you work out of —\n")
	} else {
		b.WriteString(headerStyle.Render(" Welcome to jarvis ") + "\n\n")
		b.WriteString("No config found yet. Pick the project board(s) you work out of —\n")
	}
	b.WriteString(subtitleStyle.Render("they drive the Project View. Saved to config.json; re-open anytime with P.") + "\n\n")
	b.WriteString(m.filter.View() + "\n\n")

	if len(m.matches) == 0 {
		b.WriteString(dimStyle.Render("  no projects match — clear the search to see all") + "\n")
	}
	end := m.top + onboardVisibleRows
	if end > len(m.matches) {
		end = len(m.matches)
	}
	for row := m.top; row < end; row++ {
		idx := m.matches[row]
		p := m.projects[idx]
		check := "[ ]"
		if _, ok := m.selected[idx]; ok {
			check = "[x]"
		}
		line := fmt.Sprintf("%s #%d  %s", check, p.Number, p.Title)
		if row == m.cursor {
			b.WriteString(selectedStyle.Render("▸ "+line) + "\n")
		} else {
			style := dimStyle
			if _, ok := m.selected[idx]; ok {
				style = reasonStyle // green for selected
			}
			b.WriteString("  " + style.Render(line) + "\n")
		}
	}
	if len(m.matches) > onboardVisibleRows {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  … %d of %d shown", end-m.top, len(m.matches))) + "\n")
	}

	b.WriteString("\n" + dimStyle.Render(fmt.Sprintf("%d selected", len(m.selected))) + "\n")
	b.WriteString(subtitleStyle.Render("↑/↓ move · space/tab select · enter confirm · esc cancel · type to filter") + "\n")
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

// chosenHandles returns the config handles for the selected projects, in ascending
// project-number order for a stable config file.
func (m *onboardModel) chosenHandles() []string {
	idxs := make([]int, 0, len(m.selected))
	for idx := range m.selected {
		idxs = append(idxs, idx)
	}
	sort.Slice(idxs, func(a, b int) bool {
		return m.projects[idxs[a]].Number < m.projects[idxs[b]].Number
	})
	out := make([]string, 0, len(idxs))
	for _, idx := range idxs {
		out = append(out, projectHandle(m.projects[idx].Title))
	}
	return out
}

// roleModel is the first-run role picker: a simple single-select list of the
// roles jarvis supports, used to seed `role` in a fresh config.
type roleModel struct {
	cursor    int
	confirmed bool
	cancelled bool
	width     int
}

// newRoleModel builds the role picker, landing on the role matching current (or
// the first role when current is empty/unknown).
func newRoleModel(current string) *roleModel {
	m := &roleModel{width: 100}
	want := normalizeRole(current)
	for i, r := range RoleChoices {
		if r.Role == want {
			m.cursor = i
			break
		}
	}
	return m
}

func (m *roleModel) Init() tea.Cmd { return nil }

func (m *roleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		case "up", "ctrl+p", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "ctrl+n", "j":
			if m.cursor < len(RoleChoices)-1 {
				m.cursor++
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *roleModel) View() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(" Welcome to jarvis ") + "\n\n")
	b.WriteString("What's your primary role? It tailors how Start Work seeds a Claude session.\n")
	b.WriteString(subtitleStyle.Render("Saved to config.json; override the per-role prompt there anytime.") + "\n\n")
	for i, r := range RoleChoices {
		line := fmt.Sprintf("%-10s %s", r.Label, dimStyle.Render(r.Desc))
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("▸ "+line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}
	b.WriteString("\n" + subtitleStyle.Render("↑/↓ move · enter confirm · esc cancel") + "\n")
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

// runRolePicker runs the standalone role picker and returns the chosen role.
// cancelled is true if the user aborted (esc/ctrl+c).
func runRolePicker(current string) (role string, cancelled bool, err error) {
	rm := newRoleModel(current)
	res, err := tea.NewProgram(rm, tea.WithAltScreen()).Run()
	if err != nil {
		return "", false, err
	}
	final := res.(*roleModel)
	if final.cancelled {
		return "", true, nil
	}
	return RoleChoices[final.cursor].Role, false, nil
}

// runOnboarding runs the first-run role + project pickers and writes a fresh
// config.json with the chosen role and primary projects. Returns cancelled=true if
// the user aborted (esc/ctrl+c) so the caller can exit without writing anything.
func runOnboarding(repo, configPath string) (cancelled bool, err error) {
	role, cancelled, err := runRolePicker("")
	if err != nil || cancelled {
		return cancelled, err
	}

	owner := repoOwner(repo)
	projects, err := ghapi.ListOrgProjects(owner)
	if err != nil {
		// The overwhelmingly common first-run cause is a gh token without the
		// `project` scope (gh doesn't grant it by default). Print actionable setup
		// guidance instead of the opaque "exit status 1".
		return false, reportProjectAccessFailure(owner)
	}
	open := projects[:0]
	for _, p := range projects {
		if !p.Closed {
			open = append(open, p)
		}
	}
	if len(open) == 0 {
		return false, fmt.Errorf("no open projects found for %s", owner)
	}

	om := newOnboardModel(owner, open)
	res, err := tea.NewProgram(om, tea.WithAltScreen()).Run()
	if err != nil {
		return false, err
	}
	final := res.(*onboardModel)
	if final.cancelled {
		return true, nil
	}

	cfg := &Config{Role: role, PrimaryProjects: final.chosenHandles()}
	if err := cfg.Save(configPath); err != nil {
		return false, fmt.Errorf("saving config: %w", err)
	}
	return false, nil
}

// reportProjectAccessFailure prints actionable guidance to stderr for a failed
// project listing at onboarding and returns a terse error to stop setup. It
// tailors the message: log in first if gh isn't authenticated, otherwise grant
// the `project` scope gh omits by default. The detail lives in the printed block,
// so callers should silence cobra's usage/error dump for a clean result.
func reportProjectAccessFailure(owner string) error {
	var b strings.Builder
	b.WriteString(errStyle.Render("jarvis couldn't list your GitHub projects for "+owner+".") + "\n\n")
	if !ghLoggedIn() {
		b.WriteString("You're not logged in to the GitHub CLI. Log in with project access, then run jarvis again:\n\n")
		b.WriteString("    " + reasonStyle.Render("gh auth login") + "\n")
		b.WriteString("    " + reasonStyle.Render("gh auth refresh -s project") + "\n\n")
	} else {
		b.WriteString("Your GitHub CLI login is missing the " + reasonStyle.Render("project") +
			" scope (gh doesn't grant it by default).\nAdd it, then run jarvis again:\n\n")
		b.WriteString("    " + reasonStyle.Render("gh auth refresh -s project") + "\n\n")
	}
	b.WriteString(dimStyle.Render("See the \"Setup\" section of the github-manage README for details."))
	fmt.Fprintln(os.Stderr, b.String())
	return errors.New("GitHub project access required (see instructions above)")
}

func ghLoggedIn() bool {
	_, err := ghapi.RunGH("auth", "status")
	return err == nil
}
