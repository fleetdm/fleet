package jarvis

import (
	"fmt"
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
	width     int
}

// newOnboardModel builds the picker over the given open projects.
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

// recompute refilters and reranks the project list against the current query.
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

// clampScroll keeps the cursor within the visible window.
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
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			m.confirmed = true
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
	b.WriteString(headerStyle.Render(" Welcome to jarvis ") + "\n\n")
	b.WriteString("No config found yet. Pick the project board(s) you work out of —\n")
	b.WriteString(subtitleStyle.Render("they seed the Project View. You can edit this later in config.json.") + "\n\n")
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

// runOnboarding runs the first-run project picker and writes a fresh config.json
// with the chosen primary projects. Returns cancelled=true if the user aborted
// (esc/ctrl+c) so the caller can exit without writing anything.
func runOnboarding(repo, configPath string) (cancelled bool, err error) {
	owner := repoOwner(repo)
	projects, err := ghapi.ListOrgProjects(owner)
	if err != nil {
		return false, fmt.Errorf("listing projects for %s: %w", owner, err)
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

	cfg := &Config{PrimaryProjects: final.chosenHandles()}
	if err := cfg.Save(configPath); err != nil {
		return false, fmt.Errorf("saving config: %w", err)
	}
	return false, nil
}
