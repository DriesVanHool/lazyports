package ui

import (
	"context"
	"errors"
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
	frameStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	toolbarStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	tableFrameStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	helpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	statusStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("108")).Bold(true)
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	detailStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("241")).Background(lipgloss.Color("235")).Padding(1, 2)
	searchStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	headerStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	subtitleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	selectedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("237")).Bold(true)
	emptyStateStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Italic(true)
	keyStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("223"))
	sepStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	labelStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("223"))
	valueStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	modalTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	badgeStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("223")).Background(lipgloss.Color("237")).Padding(0, 1)
	logoStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("223")).Bold(true)
	compactLogoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("223")).Bold(true)
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
	pid    int
	result ports.TerminateResult
	err    error
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
	forceKill   *ports.Entry
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
		m.resize(msg.Width, msg.Height)
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
		var needsForce ports.NeedsForceError
		if errors.As(msg.err, &needsForce) {
			if entry, ok := m.selectedEntry(); ok && entry.PID == msg.pid {
				entryCopy := entry
				m.forceKill = &entryCopy
			} else if m.confirmKill != nil && m.confirmKill.PID == msg.pid {
				entryCopy := *m.confirmKill
				m.forceKill = &entryCopy
			}
			m.confirmKill = nil
			m.lastErr = nil
			m.status = fmt.Sprintf("PID %d ignored %s; confirm force kill", msg.pid, needsForce.GracePeriod)
			return m, nil
		}
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = ports.SummarizeKillError(msg.err)
			return m, nil
		}

		m.lastErr = nil
		m.confirmKill = nil
		m.forceKill = nil
		if msg.result.Forced {
			m.status = fmt.Sprintf("Force killed PID %d with %s", msg.pid, msg.result.Signal)
		} else {
			m.status = fmt.Sprintf("Terminated PID %d with %s", msg.pid, msg.result.Signal)
		}
		if m.detail != nil && m.detail.PID == msg.pid {
			m.detail = nil
		}
		return m, loadPortsCmd()
	case tea.KeyMsg:
		if m.confirmKill != nil {
			switch msg.String() {
			case "y":
				entry := *m.confirmKill
				m.status = fmt.Sprintf("Sending %s to PID %d...", gracefulSignalLabel(), entry.PID)
				m.lastErr = nil
				return m, terminatePIDCmd(entry.PID)
			case "n", "esc":
				m.confirmKill = nil
				m.status = "Terminate cancelled"
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}

		if m.forceKill != nil {
			switch msg.String() {
			case "y":
				entry := *m.forceKill
				m.status = fmt.Sprintf("Force killing PID %d...", entry.PID)
				m.lastErr = nil
				return m, forceKillPIDCmd(entry.PID)
			case "n", "esc":
				m.forceKill = nil
				m.status = "Force kill cancelled"
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
				m.forceKill = nil
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

	banner := renderBanner(m.width, m.height)
	parts := []string{banner, renderToolbar(m.modeLabel(), len(m.filtered), m.searching)}
	if m.searching {
		parts = append(parts, toolbarStyle.Render(searchStyle.Render(m.input.View())))
	}
	if len(m.filtered) == 0 && !m.searching && m.lastErr == nil {
		parts = append(parts, emptyStateStyle.Render("No matching ports."))
	}
	parts = append(parts, tableFrameStyle.Render(m.table.View()))

	status := statusStyle.Render(m.status)
	if m.lastErr != nil {
		status = errorStyle.Render(m.lastErr.Error())
	}
	parts = append(parts, toolbarStyle.Render(status))
	parts = append(parts, renderHelp(m.width))

	base := frameStyle.Render(strings.Join(parts, "\n"))
	if m.detail != nil {
		return m.renderModal(base, renderDetail(*m.detail))
	}
	if m.confirmKill != nil {
		return m.renderModal(base, renderKillConfirm(*m.confirmKill))
	}
	if m.forceKill != nil {
		return m.renderModal(base, renderForceKillConfirm(*m.forceKill))
	}
	return base
}

func (m *model) resize(width, height int) {
	m.width = width
	m.height = height
	m.updateTableLayout()
	m.updateTableHeight()
}

func (m *model) updateTableLayout() {
	available := max(16, m.width-14)
	portWidth := 6
	pidWidth := 7
	protocolWidth := 10
	stateWidth := 14
	processWidth := max(8, available-(portWidth+pidWidth+protocolWidth+stateWidth))

	deficit := (portWidth + pidWidth + protocolWidth + stateWidth + processWidth) - available
	if deficit > 0 {
		shrink := min(deficit, stateWidth-8)
		stateWidth -= shrink
		deficit -= shrink
	}
	if deficit > 0 {
		shrink := min(deficit, protocolWidth-6)
		protocolWidth -= shrink
		deficit -= shrink
	}
	if deficit > 0 {
		shrink := min(deficit, pidWidth-4)
		pidWidth -= shrink
		deficit -= shrink
	}
	if deficit > 0 {
		shrink := min(deficit, portWidth-4)
		portWidth -= shrink
		deficit -= shrink
	}
	if deficit > 0 {
		processWidth = max(4, processWidth-deficit)
	}

	columns := m.table.Columns()
	columns[0].Width = portWidth
	columns[1].Width = processWidth
	columns[2].Width = pidWidth
	columns[3].Width = protocolWidth
	columns[4].Width = stateWidth
	m.table.SetColumns(columns)
	m.table.SetWidth(available)
}

func (m *model) updateTableHeight() {
	reserved := frameVerticalSize() + bannerHeight(m.width) + 1 + 1 + helpHeight(m.width) + tableFrameVerticalSize()
	if m.searching {
		reserved++
	}
	m.table.SetHeight(max(3, m.height-reserved))
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
			displayState(entry),
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
		renderField("State", displayState(entry)),
		renderField("Local", renderLocal(entry)),
		renderField("Remote", renderRemote(entry)),
		renderField("Summary", summarizeEntry(entry)),
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
		renderField("Local", renderLocal(entry)),
		renderField("Protocol", entry.Protocol),
		renderField("State", displayState(entry)),
		renderField("Action", fmt.Sprintf("send %s and wait briefly", gracefulSignalLabel())),
		"",
		helpStyle.Render("y sends a graceful terminate first  |  n cancels"),
	}
	return strings.Join(lines, "\n")
}

func renderForceKillConfirm(entry ports.Entry) string {
	lines := []string{
		modalTitleStyle.Render("Force Kill Required"),
		renderField("Process", entry.Process),
		renderField("PID", fmt.Sprintf("%d", entry.PID)),
		renderField("Port", fmt.Sprintf("%d", entry.Port)),
		renderField("Local", renderLocal(entry)),
		renderField("Action", fmt.Sprintf("send %s immediately", forceSignalLabel())),
		"",
		helpStyle.Render("Process did not exit gracefully. y force kills  |  n leaves it running"),
	}
	return strings.Join(lines, "\n")
}

func loadPortsCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := ports.List(context.Background(), ports.ListOptions{IncludeConnections: true})
		return loadMsg{entries: entries, err: err}
	}
}

func terminatePIDCmd(pid int) tea.Cmd {
	return func() tea.Msg {
		result, err := ports.TerminatePID(context.Background(), pid, ports.TerminateOptions{})
		return killMsg{pid: pid, result: result, err: err}
	}
}

func forceKillPIDCmd(pid int) tea.Cmd {
	return func() tea.Msg {
		result, err := ports.TerminatePID(context.Background(), pid, ports.TerminateOptions{Force: true})
		return killMsg{pid: pid, result: result, err: err}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
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

func renderLocal(entry ports.Entry) string {
	if entry.LocalAddress == "" {
		return fmt.Sprintf(":%d", entry.Port)
	}
	return fmt.Sprintf("%s:%d", entry.LocalAddress, entry.Port)
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

func summarizeEntry(entry ports.Entry) string {
	if entry.Kind == ports.KindListener {
		return fmt.Sprintf("%s listening on %s", entry.Process, renderLocal(entry))
	}
	return fmt.Sprintf("%s %s -> %s", entry.Protocol, renderLocal(entry), renderRemote(entry))
}

func gracefulSignalLabel() string {
	if strings.EqualFold(forceSignalLabel(), "KILL") {
		return "TERM"
	}
	return "TERM"
}

func forceSignalLabel() string {
	return "KILL"
}

func kindBadge(kind ports.Kind) string {
	if kind == ports.KindListener {
		return "listen"
	}
	return "conn"
}

func displayState(entry ports.Entry) string {
	state := strings.TrimSpace(entry.State)
	if state != "" {
		return state
	}
	if entry.Kind == ports.KindListener {
		return "LISTEN"
	}
	return "ESTABLISHED"
}

func renderHelp(width int) string {
	items := []string{
		renderHelpItem("/", "search"),
		renderHelpItem("a", "toggle listeners/all"),
		renderHelpItem("r", "refresh"),
		renderHelpItem("k", "terminate process on selected port"),
		renderHelpItem("Enter", "details"),
		renderHelpItem("q", "quit"),
	}
	contentWidth := max(20, width-8)
	header := labelStyle.Render("Shortcuts")
	sep := sepStyle.Render("  |  ")
	rows := []string{}
	current := header
	for _, item := range items {
		candidate := current + sep + item
		if lipgloss.Width(candidate) > contentWidth && current != header {
			rows = append(rows, current)
			current = item
			continue
		}
		if current == header {
			current = current + "  " + item
			continue
		}
		current = candidate
	}
	rows = append(rows, current)
	return toolbarStyle.Render(strings.Join(rows, "\n"))
}

func renderHelpItem(key, description string) string {
	return keyStyle.Render(key) + " " + helpStyle.Render(description)
}

func renderBanner(width, height int) string {
	logo := compactLogoStyle.Render("LAZYPORTS")
	if width >= lipgloss.Width(appTitle)+6 {
		logo = logoStyle.Render(appTitle)
	}
	lines := []string{
		logo,
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

func (m model) modalWidth() int {
	return min(72, max(24, m.width-12))
}

func (m model) renderModal(base string, content string) string {
	if m.width <= 0 || m.height <= 0 {
		return base + "\n\n" + detailStyle.Render(content)
	}

	modal := detailStyle.Width(m.modalWidth()).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func bannerHeight(width int) int {
	if width >= lipgloss.Width(appTitle)+6 {
		return 7
	}
	return 2
}

func helpHeight(width int) int {
	contentWidth := max(20, width-8)
	items := []string{
		renderHelpItem("/", "search"),
		renderHelpItem("a", "toggle listeners/all"),
		renderHelpItem("r", "refresh"),
		renderHelpItem("k", "terminate process on selected port"),
		renderHelpItem("Enter", "details"),
		renderHelpItem("q", "quit"),
	}
	headerWidth := lipgloss.Width(labelStyle.Render("Shortcuts") + "  ")
	rows := 1
	lineWidth := headerWidth
	sepWidth := lipgloss.Width(sepStyle.Render("  |  "))
	for i, item := range items {
		itemWidth := lipgloss.Width(item)
		needed := itemWidth
		if i > 0 && lineWidth > 0 {
			needed += sepWidth
		}
		if lineWidth > headerWidth && lineWidth+needed > contentWidth {
			rows++
			lineWidth = itemWidth
			continue
		}
		if lineWidth <= headerWidth {
			lineWidth += itemWidth
			continue
		}
		lineWidth += needed
	}
	return rows
}

func frameVerticalSize() int {
	return 2
}

func tableFrameVerticalSize() int {
	return 2
}
