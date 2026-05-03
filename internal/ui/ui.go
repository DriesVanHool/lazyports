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
	frameStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1, 1)
	toolbarStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	tableFrameStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	helpStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	statusStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("109"))
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	detailStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("244")).Padding(1, 2)
	searchStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	headerStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	subtitleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("237")).Bold(true)
	emptyStateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Italic(true)
	keyStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	sepStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	labelStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	valueStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	modalTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	badgeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("251")).Background(lipgloss.Color("238")).Padding(0, 1)
)

const appTitle = "██╗      █████╗ ███████╗██╗   ██╗██████╗  ██████╗ ██████╗ ████████╗███████╗\n" +
	"██║     ██╔══██╗╚══███╔╝╚██╗ ██╔╝██╔══██╗██╔═══██╗██╔══██╗╚══██╔══╝██╔════╝\n" +
	"██║     ███████║  ███╔╝  ╚████╔╝ ██████╔╝██║   ██║██████╔╝   ██║   ███████╗\n" +
	"██║     ██╔══██║ ███╔╝    ╚██╔╝  ██╔═══╝ ██║   ██║██╔══██╗   ██║   ╚════██║\n" +
	"███████╗██║  ██║███████╗   ██║   ██║     ╚██████╔╝██║  ██║   ██║   ███████║\n" +
	"╚══════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝      ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝"

type loadMsg struct {
	entries []ports.Entry
	err     error
}

type killMsg struct {
	pid int
	err error
}

type viewMode string

const (
	modeListeners viewMode = "listeners"
	modeAll       viewMode = "all"
)

type model struct {
	table       table.Model
	input       textinput.Model
	entries     []ports.Entry
	filtered    []ports.Entry
	searching   bool
	detail      *ports.Entry
	confirmKill *ports.Entry
	status      string
	lastErr     error
	width       int
	height      int
	mode        viewMode
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
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("255")).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).BorderForeground(lipgloss.Color("240"))
	styles.Cell = styles.Cell.Foreground(lipgloss.Color("252"))
	styles.Selected = selectedStyle
	t.SetStyles(styles)

	input := textinput.New()
	input.Placeholder = "Search ports, process, pid, protocol"
	input.Prompt = "/ "
	input.CharLimit = 128

	return model{
		table:  t,
		input:  input,
		status: "Loading ports...",
		mode:   modeListeners,
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
		m.table.SetHeight(max(6, msg.Height-18))
		return m, nil
	case loadMsg:
		m.lastErr = msg.err
		if msg.err != nil {
			m.status = "Refresh failed"
			return m, nil
		}

		m.entries = msg.entries
		m.status = fmt.Sprintf("Loaded %d row(s) in %s mode", len(msg.entries), m.modeLabel())
		m.applyFilter()
		return m, nil
	case killMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = ports.SummarizeKillError(msg.err)
			return m, nil
		}

		m.lastErr = nil
		m.confirmKill = nil
		m.status = fmt.Sprintf("Terminated PID %d", msg.pid)
		if m.detail != nil && m.detail.PID == msg.pid {
			m.detail = nil
		}
		return m, loadPortsCmd()
	case tea.KeyMsg:
		if m.confirmKill != nil {
			switch msg.String() {
			case "y":
				entry := *m.confirmKill
				m.status = fmt.Sprintf("Killing PID %d...", entry.PID)
				m.lastErr = nil
				return m, killPIDCmd(entry.PID)
			case "n", "esc":
				m.confirmKill = nil
				m.status = "Kill cancelled"
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}

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
			m.status = "Refreshing ports..."
			m.lastErr = nil
			return m, loadPortsCmd()
		case "a":
			m.toggleMode()
			m.status = fmt.Sprintf("Switched to %s mode", m.modeLabel())
			m.applyFilter()
			return m, nil
		case "enter":
			if entry, ok := m.selectedEntry(); ok {
				entryCopy := entry
				m.detail = &entryCopy
			}
			return m, nil
		case "k":
			if entry, ok := m.selectedEntry(); ok {
				if entry.Kind != ports.KindListener {
					m.status = "Terminate is only available for listening processes"
					return m, nil
				}
				entryCopy := entry
				m.confirmKill = &entryCopy
				m.lastErr = nil
				return m, nil
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

	banner := renderBanner()
	parts := []string{banner, renderToolbar(m.modeLabel(), len(m.filtered), m.searching)}
	if m.searching {
		parts = append(parts, toolbarStyle.Render(searchStyle.Render(m.input.View())))
	}
	if len(m.filtered) == 0 && !m.searching && m.lastErr == nil {
		parts = append(parts, emptyStateStyle.Render("No matching ports."))
	}
	parts = append(parts, tableFrameStyle.Render(m.table.View()))

	if m.detail != nil {
		parts = append(parts, detailStyle.Render(renderDetail(*m.detail)))
	}
	if m.confirmKill != nil {
		parts = append(parts, detailStyle.Render(renderKillConfirm(*m.confirmKill)))
	}

	status := statusStyle.Render(m.status)
	if m.lastErr != nil {
		status = errorStyle.Render(m.lastErr.Error())
	}
	parts = append(parts, toolbarStyle.Render(status))
	parts = append(parts, renderHelp())

	return frameStyle.Width(max(40, m.width)).Render(strings.Join(parts, "\n"))
}

func (m *model) applyFilter() {
	query := strings.TrimSpace(m.input.Value())
	visible := m.visibleEntries()
	if query == "" {
		m.filtered = append([]ports.Entry(nil), visible...)
		m.syncRows()
		return
	}

	filtered := make([]ports.Entry, 0, len(visible))
	for _, entry := range visible {
		haystack := fmt.Sprintf("%d %s %d %s %s %s %s", entry.Port, entry.Process, entry.PID, entry.Protocol, entry.State, entry.Kind, entry.LocalAddress)
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
			fmt.Sprintf("%s %s", entry.State, kindBadge(entry.Kind)),
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
	lines := []string{
		modalTitleStyle.Render("Connection Details"),
		renderField("Port", fmt.Sprintf("%d", entry.Port)),
		renderField("Process", entry.Process),
		renderField("PID", fmt.Sprintf("%d", entry.PID)),
		renderField("Protocol", entry.Protocol),
		renderField("Kind", string(entry.Kind)),
		renderField("Local", entry.LocalAddress),
		renderField("Remote", renderRemote(entry)),
		renderField("State", entry.State),
		renderField("Details", entry.Details),
		"",
		helpStyle.Render("Enter or Esc closes this panel."),
	}
	return strings.Join(lines, "\n")
}

func renderKillConfirm(entry ports.Entry) string {
	lines := []string{
		modalTitleStyle.Render("Terminate Process"),
		renderField("Process", entry.Process),
		renderField("PID", fmt.Sprintf("%d", entry.PID)),
		renderField("Port", fmt.Sprintf("%d", entry.Port)),
		"",
		helpStyle.Render("y confirms termination  |  n cancels"),
	}
	return strings.Join(lines, "\n")
}

func loadPortsCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := ports.List(context.Background(), ports.ListOptions{IncludeConnections: true})
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

func (m model) visibleEntries() []ports.Entry {
	if m.mode == modeAll {
		return m.entries
	}
	visible := make([]ports.Entry, 0, len(m.entries))
	for _, entry := range m.entries {
		if entry.Kind == ports.KindListener {
			visible = append(visible, entry)
		}
	}
	return visible
}

func (m *model) toggleMode() {
	if m.mode == modeListeners {
		m.mode = modeAll
		return
	}
	m.mode = modeListeners
}

func (m model) modeLabel() string {
	if m.mode == modeAll {
		return "all connections"
	}
	return "listeners"
}

func renderRemote(entry ports.Entry) string {
	if entry.RemoteAddress == "" {
		return "-"
	}
	if entry.RemotePort > 0 {
		return fmt.Sprintf("%s:%d", entry.RemoteAddress, entry.RemotePort)
	}
	return entry.RemoteAddress
}

func kindBadge(kind ports.Kind) string {
	if kind == ports.KindListener {
		return "listen"
	}
	return "conn"
}

func renderHelp() string {
	items := []string{
		renderHelpItem("/", "search"),
		renderHelpItem("a", "toggle listeners/all"),
		renderHelpItem("r", "refresh"),
		renderHelpItem("k", "terminate process on selected port"),
		renderHelpItem("Enter", "details"),
		renderHelpItem("q", "quit"),
	}
	return toolbarStyle.Render(labelStyle.Render("Shortcuts") + "  " + strings.Join(items, sepStyle.Render("  |  ")))
}

func renderHelpItem(key, description string) string {
	return keyStyle.Render(key) + " " + helpStyle.Render(description)
}

func renderBanner() string {
	lines := []string{
		headerStyle.Render(appTitle),
		subtitleStyle.Render("CLI and TUI port inspector"),
	}
	return strings.Join(lines, "\n")
}

func renderToolbar(mode string, count int, searching bool) string {
	items := []string{
		badgeStyle.Render("MODE " + strings.ToUpper(mode)),
		badgeStyle.Render(fmt.Sprintf("ROWS %d", count)),
	}
	if searching {
		items = append(items, badgeStyle.Render("SEARCH"))
	}
	return toolbarStyle.Render(strings.Join(items, "  "))
}

func renderField(label, value string) string {
	return labelStyle.Render(label+":") + " " + valueStyle.Render(value)
}
