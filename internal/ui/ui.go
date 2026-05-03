package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/DriesVanHool/lazyports/internal/ports"
)

var (
	frameStyle      = lipgloss.NewStyle().Padding(1, 2)
	helpStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	statusStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	detailStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("111")).Padding(1, 2)
	searchStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))
	headerStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	taglineStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("99")).Bold(true)
	emptyStateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("249")).Italic(true)
)

type loadMsg struct {
	entries []ports.Entry
	err     error
}

type killMsg struct {
	pid int
	err error
}

type model struct {
	table     table.Model
	input     textinput.Model
	entries   []ports.Entry
	filtered  []ports.Entry
	searching bool
	detail    *ports.Entry
	status    string
	lastErr   error
	width     int
	height    int
}

func Run() error {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newModel() model {
	columns := []table.Column{
		{Title: "Port", Width: 8},
		{Title: "Process", Width: 24},
		{Title: "PID", Width: 8},
		{Title: "Protocol", Width: 10},
		{Title: "State", Width: 14},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(nil),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("213"))
	styles.Selected = selectedStyle
	t.SetStyles(styles)

	input := textinput.New()
	input.Placeholder = "Search ports, process, pid, protocol"
	input.Prompt = "/ "
	input.CharLimit = 128

	return model{
		table:  t,
		input:  input,
		status: "Waking up the port gremlins...",
	}
}

func (m model) Init() tea.Cmd {
	return loadPortsCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(max(5, msg.Height-9))
		return m, nil
	case loadMsg:
		m.lastErr = msg.err
		if msg.err != nil {
			m.status = "Refresh failed"
			return m, nil
		}

		m.entries = msg.entries
		m.status = fmt.Sprintf("Found %d port binding(s)", len(msg.entries))
		m.applyFilter()
		return m, nil
	case killMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = fmt.Sprintf("Kill failed for PID %d", msg.pid)
			return m, nil
		}

		m.lastErr = nil
		m.status = fmt.Sprintf("PID %d has left the chat", msg.pid)
		if m.detail != nil && m.detail.PID == msg.pid {
			m.detail = nil
		}
		return m, loadPortsCmd()
	case tea.KeyMsg:
		if m.detail != nil {
			switch msg.String() {
			case "enter", "esc":
				m.detail = nil
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}

		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.input.Blur()
				m.input.SetValue("")
				m.applyFilter()
				return m, nil
			case "enter":
				m.searching = false
				m.input.Blur()
				m.applyFilter()
				return m, nil
			}

			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.applyFilter()
			return m, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "/":
			m.searching = true
			m.input.Focus()
			return m, textinput.Blink
		case "r":
			m.status = "Refreshing the port party..."
			m.lastErr = nil
			return m, loadPortsCmd()
		case "enter":
			if entry, ok := m.selectedEntry(); ok {
				entryCopy := entry
				m.detail = &entryCopy
			}
			return m, nil
		case "k":
			if entry, ok := m.selectedEntry(); ok {
				m.status = fmt.Sprintf("Killing PID %d...", entry.PID)
				m.lastErr = nil
				return m, killPIDCmd(entry.PID)
			}
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := lipgloss.JoinHorizontal(lipgloss.Top, headerStyle.Render("LazyPorts"), taglineStyle.Render("  find ports, keep it chill"))
	parts := []string{title}
	if m.searching {
		parts = append(parts, searchStyle.Render(m.input.View()))
	}
	if len(m.filtered) == 0 && !m.searching && m.lastErr == nil {
		parts = append(parts, emptyStateStyle.Render("No matching ports right now. Very peaceful."))
	}
	parts = append(parts, m.table.View())

	if m.detail != nil {
		parts = append(parts, detailStyle.Render(renderDetail(*m.detail)))
	}

	status := statusStyle.Render(m.status)
	if m.lastErr != nil {
		status = errorStyle.Render(m.lastErr.Error())
	}
	parts = append(parts, status)
	parts = append(parts, helpStyle.Render("/ search  r refresh  k kick selected process  Enter details  q quit"))

	return frameStyle.Width(max(40, m.width)).Render(strings.Join(parts, "\n"))
}

func (m *model) applyFilter() {
	query := strings.TrimSpace(m.input.Value())
	if query == "" {
		m.filtered = append([]ports.Entry(nil), m.entries...)
		m.syncRows()
		return
	}

	filtered := make([]ports.Entry, 0, len(m.entries))
	for _, entry := range m.entries {
		haystack := fmt.Sprintf("%d %s %d %s %s", entry.Port, entry.Process, entry.PID, entry.Protocol, entry.State)
		if fuzzy.MatchNormalizedFold(query, haystack) {
			filtered = append(filtered, entry)
		}
	}

	m.filtered = filtered
	m.syncRows()
	m.status = fmt.Sprintf("Filtered down to %d result(s)", len(filtered))
}

func (m *model) syncRows() {
	rows := make([]table.Row, 0, len(m.filtered))
	for _, entry := range m.filtered {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", entry.Port),
			entry.Process,
			fmt.Sprintf("%d", entry.PID),
			entry.Protocol,
			entry.State,
		})
	}

	m.table.SetRows(rows)
	if len(rows) == 0 {
		m.table.SetCursor(0)
		return
	}
	if m.table.Cursor() >= len(rows) {
		m.table.SetCursor(len(rows) - 1)
	}
}

func (m model) selectedEntry() (ports.Entry, bool) {
	if len(m.filtered) == 0 {
		return ports.Entry{}, false
	}

	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.filtered) {
		return ports.Entry{}, false
	}

	return m.filtered[idx], true
}

func renderDetail(entry ports.Entry) string {
	return fmt.Sprintf(
		"Port: %d\nProcess: %s\nPID: %d\nProtocol: %s\nState: %s\nDetails: %s\n\nPress Enter or Esc to slide back.",
		entry.Port,
		entry.Process,
		entry.PID,
		entry.Protocol,
		entry.State,
		entry.Details,
	)
}

func loadPortsCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := ports.List(context.Background())
		return loadMsg{entries: entries, err: err}
	}
}

func killPIDCmd(pid int) tea.Cmd {
	return func() tea.Msg {
		return killMsg{pid: pid, err: ports.KillPID(pid)}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
