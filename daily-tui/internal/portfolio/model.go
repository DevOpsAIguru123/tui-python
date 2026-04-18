package portfolio

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Section selects which list the user is currently viewing.
type Section int

const (
	SectionHoldings Section = iota
	SectionWatchlist
)

// editMode tracks what the single text input represents at any given moment.
type editMode int

const (
	modeIdle editMode = iota
	modeAddHolding
	modeEditHolding
	modeUpdatePrice
	modeAddWatch
	modeEditWatch
)

// Model is the Bubble Tea model for the Portfolio tab.
// It owns a multi-step input form: the user enters one field, presses Enter,
// the input is cleared and reconfigured for the next field. This keeps the
// UI simple (one text widget) while still accepting structured data.
type Model struct {
	store       *Store
	section     Section
	holdingsCur int
	watchCur    int
	mode        editMode
	fieldIdx    int
	fields      []string
	input       textinput.Model
	editIdx     int
	err         string
}

// New creates a PortfolioModel backed by the given store.
func New(s *Store) Model {
	ti := textinput.New()
	ti.CharLimit = 120
	return Model{store: s, input: ti}
}

// Inputting reports whether the model owns all keys (during multi-step entry).
func (m Model) Inputting() bool { return m.mode != modeIdle }

// Section returns the currently active section (exported for tests).
func (m Model) Section() Section { return m.section }

// Cursor returns the cursor position for the active section (exported for tests).
func (m Model) Cursor() int {
	if m.section == SectionHoldings {
		return m.holdingsCur
	}
	return m.watchCur
}

// Count returns the number of entries in the active section (sidebar badge).
func (m Model) Count() int {
	if m.section == SectionHoldings {
		return len(m.store.Holdings())
	}
	return len(m.store.Watchlist())
}

// Title is the subtitle shown in the breadcrumb header. It reports the
// active section and its row count so the breadcrumb is informative rather
// than a static label.
func (m Model) Title() string {
	if m.section == SectionHoldings {
		n := len(m.store.Holdings())
		if n == 0 {
			return "holdings · empty"
		}
		return fmt.Sprintf("holdings · %d", n)
	}
	n := len(m.store.Watchlist())
	if n == 0 {
		return "watchlist · empty"
	}
	return fmt.Sprintf("watchlist · %d", n)
}

// Help returns the key hints rendered by the help bar.
func (m Model) Help() []theme.KeyHint {
	if m.Inputting() {
		return []theme.KeyHint{
			{Key: "enter", Label: "next/confirm"},
			{Key: "esc", Label: "cancel"},
		}
	}
	if m.section == SectionHoldings {
		return []theme.KeyHint{
			{Key: "↑↓", Label: "navigate"},
			{Key: "a", Label: "add"},
			{Key: "e", Label: "edit"},
			{Key: "d", Label: "delete"},
			{Key: "p", Label: "price"},
			{Key: "2", Label: "watchlist"},
		}
	}
	return []theme.KeyHint{
		{Key: "↑↓", Label: "navigate"},
		{Key: "a", Label: "add"},
		{Key: "e", Label: "edit"},
		{Key: "d", Label: "delete"},
		{Key: "1", Label: "holdings"},
	}
}

// Init is a no-op.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	if m.Inputting() {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.Inputting() {
		return m.handleFormKey(msg)
	}
	return m.handleBrowseKey(msg)
}

// handleBrowseKey processes keys when no input form is open.
func (m Model) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.section == SectionHoldings && m.holdingsCur > 0 {
			m.holdingsCur--
		}
		if m.section == SectionWatchlist && m.watchCur > 0 {
			m.watchCur--
		}
	case tea.KeyDown:
		if m.section == SectionHoldings && m.holdingsCur < len(m.store.Holdings())-1 {
			m.holdingsCur++
		}
		if m.section == SectionWatchlist && m.watchCur < len(m.store.Watchlist())-1 {
			m.watchCur++
		}
	case tea.KeyRunes:
		return m.handleBrowseRune(string(msg.Runes))
	}
	return m, nil
}

func (m Model) handleBrowseRune(r string) (tea.Model, tea.Cmd) {
	switch r {
	case "1":
		m.section = SectionHoldings
	case "2":
		m.section = SectionWatchlist
	case "a":
		m.startAdd()
	case "e":
		m.startEdit()
	case "d":
		m.deleteSelected()
	case "p":
		if m.section == SectionHoldings {
			m.startUpdatePrice()
		}
	}
	return m, nil
}

// startAdd opens the add-form for the active section.
func (m *Model) startAdd() {
	m.err = ""
	m.fieldIdx = 0
	m.fields = nil
	if m.section == SectionHoldings {
		m.mode = modeAddHolding
	} else {
		m.mode = modeAddWatch
	}
	m.configureInputForField()
}

// startEdit opens the edit-form for the selected row, pre-populating values.
func (m *Model) startEdit() {
	m.err = ""
	m.fieldIdx = 0
	if m.section == SectionHoldings {
		list := m.store.Holdings()
		if len(list) == 0 {
			return
		}
		h := list[m.holdingsCur]
		m.editIdx = m.holdingsCur
		m.fields = []string{
			h.Ticker,
			formatFloat(h.Shares),
			formatFloat(h.AvgCost),
			formatFloatOrBlank(h.LastPrice),
			h.Note,
		}
		m.mode = modeEditHolding
	} else {
		list := m.store.Watchlist()
		if len(list) == 0 {
			return
		}
		w := list[m.watchCur]
		m.editIdx = m.watchCur
		m.fields = []string{w.Ticker, w.Note}
		m.mode = modeEditWatch
	}
	m.configureInputForField()
}

// startUpdatePrice opens a single-field prompt for LastPrice.
func (m *Model) startUpdatePrice() {
	list := m.store.Holdings()
	if len(list) == 0 {
		return
	}
	m.err = ""
	m.editIdx = m.holdingsCur
	m.fieldIdx = 0
	m.fields = []string{formatFloatOrBlank(list[m.holdingsCur].LastPrice)}
	m.mode = modeUpdatePrice
	m.configureInputForField()
}

// deleteSelected removes the selected row from the active section.
func (m *Model) deleteSelected() {
	if m.section == SectionHoldings {
		list := m.store.Holdings()
		if len(list) == 0 {
			return
		}
		m.store.DeleteHolding(m.holdingsCur)
		if m.holdingsCur >= len(m.store.Holdings()) && m.holdingsCur > 0 {
			m.holdingsCur--
		}
	} else {
		list := m.store.Watchlist()
		if len(list) == 0 {
			return
		}
		m.store.DeleteWatch(m.watchCur)
		if m.watchCur >= len(m.store.Watchlist()) && m.watchCur > 0 {
			m.watchCur--
		}
	}
}

// handleFormKey drives the multi-step input form.
// Enter advances to the next field or finalises the entry.
// Esc cancels and returns to browsing.
func (m Model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.cancelForm()
		return m, nil
	case tea.KeyEnter:
		return m.commitField()
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *Model) cancelForm() {
	m.mode = modeIdle
	m.fieldIdx = 0
	m.fields = nil
	m.err = ""
	m.input.Blur()
	m.input.SetValue("")
}

// commitField stores the current input value at fieldIdx, advances, and
// either reconfigures the input for the next field or finalises the entry.
func (m Model) commitField() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.input.Value())
	spec := m.formSpec()
	fieldCount := len(spec)

	// Grow fields slice if this is an add-flow (it starts at len 0).
	for len(m.fields) <= m.fieldIdx {
		m.fields = append(m.fields, "")
	}
	m.fields[m.fieldIdx] = value

	m.fieldIdx++
	if m.fieldIdx < fieldCount {
		m.configureInputForField()
		return m, nil
	}

	// All fields collected — validate and commit.
	if err := m.finalize(); err != nil {
		// Stay in the form so the user can fix the bad field.
		m.err = err.Error()
		m.fieldIdx = 0
		m.configureInputForField()
		return m, nil
	}
	m.cancelForm()
	return m, nil
}

// finalize parses fields into a typed entry, validates, and writes to the store.
// Returns an error (with user-readable message) if validation fails.
func (m Model) finalize() error {
	switch m.mode {
	case modeAddHolding, modeEditHolding:
		h, err := parseHoldingFields(m.fields)
		if err != nil {
			return err
		}
		if m.mode == modeAddHolding {
			m.store.AddHolding(h)
		} else {
			m.store.UpdateHolding(m.editIdx, h)
		}
	case modeUpdatePrice:
		price, err := strconv.ParseFloat(m.fields[0], 64)
		if err != nil || price < 0 {
			return fmt.Errorf("price must be a non-negative number")
		}
		m.store.SetLastPrice(m.editIdx, price)
	case modeAddWatch, modeEditWatch:
		w, err := parseWatchFields(m.fields)
		if err != nil {
			return err
		}
		if m.mode == modeAddWatch {
			m.store.AddWatch(w)
		} else {
			m.store.UpdateWatch(m.editIdx, w)
		}
	}
	return nil
}

// formSpec returns labels + placeholders for the current mode's fields.
type fieldSpec struct{ Label, Placeholder string }

func (m Model) formSpec() []fieldSpec {
	switch m.mode {
	case modeAddHolding, modeEditHolding:
		return []fieldSpec{
			{"Ticker", "AAPL"},
			{"Shares", "25"},
			{"Avg cost", "182.40"},
			{"Last price (optional)", "207.15"},
			{"Note (optional)", ""},
		}
	case modeUpdatePrice:
		return []fieldSpec{{"Last price", "207.15"}}
	case modeAddWatch, modeEditWatch:
		return []fieldSpec{
			{"Ticker", "NVDA"},
			{"Note (optional)", "earnings next week"},
		}
	}
	return nil
}

// configureInputForField resets the text widget for the current fieldIdx,
// pre-populating from m.fields so edit-flows show the existing value.
func (m *Model) configureInputForField() {
	spec := m.formSpec()
	if m.fieldIdx >= len(spec) {
		return
	}
	s := spec[m.fieldIdx]
	m.input.Placeholder = s.Placeholder
	m.input.SetValue("")
	if m.fieldIdx < len(m.fields) {
		m.input.SetValue(m.fields[m.fieldIdx])
	}
	m.input.Focus()
}

// ---- View ----

// View renders the Portfolio tab.
func (m Model) View() string {
	var sb strings.Builder
	sb.WriteString(m.renderSectionTabs())
	sb.WriteString("\n\n")

	if m.section == SectionHoldings {
		sb.WriteString(m.renderHoldings())
	} else {
		sb.WriteString(m.renderWatchlist())
	}

	if m.Inputting() {
		sb.WriteString("\n")
		sb.WriteString(m.renderForm())
	}
	if m.err != "" && !m.Inputting() {
		sb.WriteString("\n")
		sb.WriteString(theme.StatusErr.Render("  " + m.err))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.RenderKeyHints(m.Help()))
	return sb.String()
}

func (m Model) renderSectionTabs() string {
	labels := []string{"Holdings", "Watchlist"}
	active := int(m.section)
	parts := make([]string, len(labels))
	for i, l := range labels {
		if i == active {
			parts[i] = theme.ActiveTab.Render(l)
		} else {
			parts[i] = theme.InactiveTab.Render(l)
		}
	}
	return strings.Join(parts, " ")
}

func (m Model) renderHoldings() string {
	var sb strings.Builder
	holdings := m.store.Holdings()

	if len(holdings) == 0 {
		sb.WriteString(theme.Dimmed.Render("  No holdings yet — press 'a' to add one"))
		sb.WriteString("\n")
		return sb.String()
	}

	// P/L can run wide (e.g. "+$3263.01 (+141.58%)") so we give it a 22-col
	// slot; Note is clipped to keep a single line from overflowing the pane.
	header := fmt.Sprintf("  %-6s %-7s %-10s %-10s %-11s %-22s %s",
		"Ticker", "Shares", "Avg Cost", "Last", "Mkt Value", "P/L", "Note")
	sb.WriteString(theme.Dimmed.Render(header))
	sb.WriteString("\n")

	for i, h := range holdings {
		line := fmt.Sprintf("%-6s %-7s %-10s %-10s %-11s %-22s %s",
			h.Ticker,
			formatFloat(h.Shares),
			"$"+formatFloat(h.AvgCost),
			formatPriceOrDash(h),
			formatMktValueOrDash(h),
			formatPLOrDash(h),
			truncate(h.Note, 12),
		)
		if i == m.holdingsCur && !m.Inputting() {
			sb.WriteString(theme.Selected.Render("▸ " + line))
		} else {
			sb.WriteString(theme.Normal.Render("  " + line))
		}
		sb.WriteString("\n")
	}

	// Totals footer
	cb, mv, pl, hasAny := m.store.Totals()
	sb.WriteString("\n")
	sb.WriteString(theme.Dimmed.Render("─ Total ") +
		theme.Dimmed.Render("──────────────────────────────────────────────────"))
	sb.WriteString("\n")
	summary := fmt.Sprintf("  Cost basis: $%s", formatFloat(cb))
	if hasAny {
		summary += fmt.Sprintf("   Market value: $%s   P/L: %s", formatFloat(mv), formatSignedDollar(pl))
	}
	sb.WriteString(theme.Normal.Render(summary))
	sb.WriteString("\n")
	return sb.String()
}

func (m Model) renderWatchlist() string {
	var sb strings.Builder
	watch := m.store.Watchlist()

	if len(watch) == 0 {
		sb.WriteString(theme.Dimmed.Render("  No watchlist entries yet — press 'a' to add one"))
		sb.WriteString("\n")
		return sb.String()
	}

	header := fmt.Sprintf("  %-8s %s", "Ticker", "Note")
	sb.WriteString(theme.Dimmed.Render(header))
	sb.WriteString("\n")

	for i, w := range watch {
		line := fmt.Sprintf("%-8s %s", w.Ticker, w.Note)
		if i == m.watchCur && !m.Inputting() {
			sb.WriteString(theme.Selected.Render("▸ " + line))
		} else {
			sb.WriteString(theme.Normal.Render("  " + line))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (m Model) renderForm() string {
	spec := m.formSpec()
	if m.fieldIdx >= len(spec) {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(theme.Dimmed.Render(m.formTitle()))
	sb.WriteString("\n")
	// Show already-entered fields as read-only context.
	for i := 0; i < m.fieldIdx; i++ {
		sb.WriteString(theme.Dimmed.Render("  " + spec[i].Label + ": "))
		sb.WriteString(theme.Normal.Render(m.fields[i]))
		sb.WriteString("\n")
	}
	sb.WriteString(theme.Dimmed.Render("  " + spec[m.fieldIdx].Label + ": "))
	sb.WriteString(m.input.View())
	sb.WriteString("\n")
	if m.err != "" {
		sb.WriteString(theme.StatusErr.Render("  " + m.err))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (m Model) formTitle() string {
	switch m.mode {
	case modeAddHolding:
		return "Adding holding:"
	case modeEditHolding:
		return "Editing holding:"
	case modeUpdatePrice:
		return "Update last price:"
	case modeAddWatch:
		return "Adding watchlist entry:"
	case modeEditWatch:
		return "Editing watchlist entry:"
	}
	return ""
}

// ---- field parsing / formatting ----

func parseHoldingFields(f []string) (Holding, error) {
	if len(f) < 5 {
		return Holding{}, fmt.Errorf("missing fields")
	}
	ticker := strings.ToUpper(strings.TrimSpace(f[0]))
	if ticker == "" {
		return Holding{}, fmt.Errorf("ticker is required")
	}
	shares, err := strconv.ParseFloat(strings.TrimSpace(f[1]), 64)
	if err != nil || shares <= 0 {
		return Holding{}, fmt.Errorf("shares must be a positive number")
	}
	avg, err := strconv.ParseFloat(strings.TrimSpace(f[2]), 64)
	if err != nil || avg < 0 {
		return Holding{}, fmt.Errorf("avg cost must be a non-negative number")
	}
	var last float64
	if s := strings.TrimSpace(f[3]); s != "" {
		last, err = strconv.ParseFloat(s, 64)
		if err != nil || last < 0 {
			return Holding{}, fmt.Errorf("last price must be a non-negative number or blank")
		}
	}
	return Holding{
		Ticker:    ticker,
		Shares:    shares,
		AvgCost:   avg,
		LastPrice: last,
		Note:      strings.TrimSpace(f[4]),
	}, nil
}

func parseWatchFields(f []string) (WatchItem, error) {
	if len(f) < 2 {
		return WatchItem{}, fmt.Errorf("missing fields")
	}
	ticker := strings.ToUpper(strings.TrimSpace(f[0]))
	if ticker == "" {
		return WatchItem{}, fmt.Errorf("ticker is required")
	}
	return WatchItem{Ticker: ticker, Note: strings.TrimSpace(f[1])}, nil
}

// truncate clips s to at most max runes, appending an ellipsis when it had
// to cut anything. Used to keep long Note / Location fields from pushing the
// row past the pane's right edge.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(runes[:max-1]) + "…"
}

// formatFloat prints a number without trailing zeros, up to 2 decimals.
func formatFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	// Drop trailing zeros after decimal to avoid "25.00" for integers.
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

func formatFloatOrBlank(v float64) string {
	if v == 0 {
		return ""
	}
	return formatFloat(v)
}

func formatPriceOrDash(h Holding) string {
	if !h.HasPrice() {
		return "—"
	}
	return "$" + formatFloat(h.LastPrice)
}

func formatMktValueOrDash(h Holding) string {
	if !h.HasPrice() {
		return "—"
	}
	return "$" + formatFloat(h.MarketValue())
}

func formatPLOrDash(h Holding) string {
	if !h.HasPrice() {
		return "—"
	}
	return fmt.Sprintf("%s (%s%%)", formatSignedDollar(h.PL()), formatSigned(h.PLPct()))
}

func formatSignedDollar(v float64) string {
	sign := "+"
	if v < 0 {
		sign = "-"
		v = -v
	}
	return sign + "$" + formatFloat(v)
}

func formatSigned(v float64) string {
	if v >= 0 {
		return "+" + formatFloat(v)
	}
	return formatFloat(v)
}
