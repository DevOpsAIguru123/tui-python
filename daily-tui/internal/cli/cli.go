// Package cli exposes each TUI tab's actions as a non-interactive CLI
// subcommand, so the same data can be scripted or queried from a shell
// without launching the Bubble Tea UI.
package cli

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/calendar"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/portfolio"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/todo"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/wifi"
)

// Known top-level subcommands. When os.Args[1] matches one of these,
// main routes to the CLI dispatcher instead of launching the TUI.
var subcommands = map[string]bool{
	"todo":      true,
	"portfolio": true,
	"wifi":      true,
	"calendar":  true,
	"actions":   true,
}

// IsSubcommand reports whether arg is a CLI subcommand (not a TUI launch).
func IsSubcommand(arg string) bool { return subcommands[arg] }

// Env bundles the external dependencies each subcommand needs. Tests
// stub the shell runner and clock to avoid actually touching WiFi or
// the current time.
type Env struct {
	ConfigDir string
	Stdout    io.Writer
	Stderr    io.Writer
	Stdin     io.Reader
	// Runner executes an external command and returns combined output.
	// Defaults to exec.Command(...).CombinedOutput(); overridden in tests.
	Runner func(name string, args ...string) ([]byte, error)
	Now    func() time.Time
	// PriceFetcher resolves live market prices for `portfolio refresh`.
	// Defaults to portfolio.NewYahooFetcher(); stubbed in tests.
	PriceFetcher portfolio.Fetcher
}

// Run dispatches args (os.Args[1:]) to the matching subcommand and
// returns the process exit code. args[0] is the subcommand name.
func Run(env Env, args []string) int {
	if env.Runner == nil {
		env.Runner = func(name string, a ...string) ([]byte, error) {
			return exec.Command(name, a...).CombinedOutput()
		}
	}
	if env.Now == nil {
		env.Now = time.Now
	}
	if len(args) == 0 {
		return usage(env.Stderr)
	}
	switch args[0] {
	case "todo":
		return runTodo(env, args[1:])
	case "portfolio":
		return runPortfolio(env, args[1:])
	case "wifi":
		return runWifi(env, args[1:])
	case "calendar":
		return runCalendar(env, args[1:])
	case "actions":
		return listActions(env.Stdout)
	}
	return usage(env.Stderr)
}

func usage(w io.Writer) int {
	fmt.Fprintln(w, "daily-tui — terminal dashboard for WiFi & tasks")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  daily-tui                      Launch the TUI")
	fmt.Fprintln(w, "  daily-tui --version | --help")
	fmt.Fprintln(w, "  daily-tui actions              List every CLI action")
	fmt.Fprintln(w, "  daily-tui todo      <cmd>      See: daily-tui todo")
	fmt.Fprintln(w, "  daily-tui portfolio <cmd>      See: daily-tui portfolio")
	fmt.Fprintln(w, "  daily-tui wifi      <cmd>      See: daily-tui wifi")
	fmt.Fprintln(w, "  daily-tui calendar  <cmd>      See: daily-tui calendar")
	return 1
}

func listActions(w io.Writer) int {
	lines := []string{
		"todo list [--all|--pending|--done]",
		"todo add <text>",
		"todo done <id>",
		"todo rm <id>",
		"portfolio list",
		"portfolio watch",
		"portfolio add <ticker> <shares> <avg_cost> [last_price] [note]",
		"portfolio price <ticker> <last_price>",
		"portfolio refresh [ticker]",
		"portfolio rm <ticker>",
		"portfolio watch-add <ticker> [note]",
		"portfolio watch-rm <ticker>",
		"wifi list",
		"wifi current",
		"wifi connect <ssid> [password]",
		"wifi forget <ssid>",
		"calendar list [days]",
	}
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
	return 0
}

// ---------- todo ----------

func runTodo(env Env, args []string) int {
	store := todo.NewStore(filepath.Join(env.ConfigDir, "todos.json"))
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "todo: list | add <text> | done <id> | rm <id>")
		return 1
	}
	switch args[0] {
	case "list":
		return todoList(env.Stdout, store, args[1:])
	case "add":
		if len(args) < 2 {
			fmt.Fprintln(env.Stderr, "todo add: missing text")
			return 1
		}
		text := strings.Join(args[1:], " ")
		t := store.Add(text)
		fmt.Fprintf(env.Stdout, "added  %s  %s\n", t.ID, t.Text)
	case "done":
		if len(args) != 2 {
			fmt.Fprintln(env.Stderr, "todo done: usage: done <id>")
			return 1
		}
		if !hasTask(store, args[1]) {
			fmt.Fprintf(env.Stderr, "todo done: no task with id %q\n", args[1])
			return 1
		}
		store.Toggle(args[1])
		fmt.Fprintln(env.Stdout, "ok")
	case "rm":
		if len(args) != 2 {
			fmt.Fprintln(env.Stderr, "todo rm: usage: rm <id>")
			return 1
		}
		if !hasTask(store, args[1]) {
			fmt.Fprintf(env.Stderr, "todo rm: no task with id %q\n", args[1])
			return 1
		}
		store.Delete(args[1])
		fmt.Fprintln(env.Stdout, "ok")
	default:
		fmt.Fprintf(env.Stderr, "todo: unknown subcommand %q\n", args[0])
		return 1
	}
	return 0
}

func hasTask(s *todo.Store, id string) bool {
	for _, t := range s.All() {
		if t.ID == id {
			return true
		}
	}
	return false
}

func todoList(w io.Writer, s *todo.Store, args []string) int {
	filter := "all"
	if len(args) > 0 {
		switch args[0] {
		case "--all":
			filter = "all"
		case "--pending":
			filter = "pending"
		case "--done":
			filter = "done"
		default:
			fmt.Fprintf(w, "todo list: unknown flag %q\n", args[0])
			return 1
		}
	}
	tasks := s.All()
	if len(tasks) == 0 {
		fmt.Fprintln(w, "(no tasks)")
		return 0
	}
	for _, t := range tasks {
		if filter == "pending" && t.Done {
			continue
		}
		if filter == "done" && !t.Done {
			continue
		}
		mark := "[ ]"
		if t.Done {
			mark = "[x]"
		}
		fmt.Fprintf(w, "%s  %s  %s\n", mark, t.ID, t.Text)
	}
	return 0
}

// ---------- portfolio ----------

func runPortfolio(env Env, args []string) int {
	store := portfolio.NewStore(filepath.Join(env.ConfigDir, "portfolio.json"))
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "portfolio: list | watch | add | price | refresh | rm | watch-add | watch-rm")
		return 1
	}
	switch args[0] {
	case "list":
		return portfolioList(env.Stdout, store)
	case "watch":
		return portfolioWatch(env.Stdout, store)
	case "add":
		return portfolioAdd(env, store, args[1:])
	case "price":
		return portfolioPrice(env, store, args[1:])
	case "refresh":
		return portfolioRefresh(env, store, args[1:])
	case "rm":
		return portfolioRm(env, store, args[1:])
	case "watch-add":
		return portfolioWatchAdd(env, store, args[1:])
	case "watch-rm":
		return portfolioWatchRm(env, store, args[1:])
	default:
		fmt.Fprintf(env.Stderr, "portfolio: unknown subcommand %q\n", args[0])
		return 1
	}
}

func portfolioList(w io.Writer, s *portfolio.Store) int {
	h := s.Holdings()
	if len(h) == 0 {
		fmt.Fprintln(w, "(no holdings)")
		return 0
	}
	fmt.Fprintf(w, "%-8s %-10s %-10s %-10s %s\n", "TICKER", "SHARES", "AVG_COST", "LAST", "NOTE")
	for _, row := range h {
		last := "—"
		if row.HasPrice() {
			last = strconv.FormatFloat(row.LastPrice, 'f', -1, 64)
		}
		fmt.Fprintf(w, "%-8s %-10s %-10s %-10s %s\n",
			row.Ticker,
			strconv.FormatFloat(row.Shares, 'f', -1, 64),
			strconv.FormatFloat(row.AvgCost, 'f', -1, 64),
			last,
			row.Note,
		)
	}
	cb, mv, pl, hasAny := s.Totals()
	fmt.Fprintf(w, "cost_basis=%s", strconv.FormatFloat(cb, 'f', 2, 64))
	if hasAny {
		fmt.Fprintf(w, " market_value=%s pl=%s",
			strconv.FormatFloat(mv, 'f', 2, 64),
			strconv.FormatFloat(pl, 'f', 2, 64))
	}
	fmt.Fprintln(w)
	return 0
}

func portfolioWatch(w io.Writer, s *portfolio.Store) int {
	items := s.Watchlist()
	if len(items) == 0 {
		fmt.Fprintln(w, "(no watchlist entries)")
		return 0
	}
	fmt.Fprintf(w, "%-8s %s\n", "TICKER", "NOTE")
	for _, it := range items {
		fmt.Fprintf(w, "%-8s %s\n", it.Ticker, it.Note)
	}
	return 0
}

func portfolioAdd(env Env, s *portfolio.Store, args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(env.Stderr, "portfolio add: usage: add <ticker> <shares> <avg_cost> [last_price] [note]")
		return 1
	}
	shares, err := strconv.ParseFloat(args[1], 64)
	if err != nil || shares <= 0 {
		fmt.Fprintln(env.Stderr, "portfolio add: shares must be a positive number")
		return 1
	}
	avg, err := strconv.ParseFloat(args[2], 64)
	if err != nil || avg < 0 {
		fmt.Fprintln(env.Stderr, "portfolio add: avg_cost must be a non-negative number")
		return 1
	}
	h := portfolio.Holding{
		Ticker:  strings.ToUpper(args[0]),
		Shares:  shares,
		AvgCost: avg,
	}
	if len(args) >= 4 && args[3] != "" {
		last, err := strconv.ParseFloat(args[3], 64)
		if err != nil || last < 0 {
			fmt.Fprintln(env.Stderr, "portfolio add: last_price must be a non-negative number")
			return 1
		}
		h.LastPrice = last
	}
	if len(args) >= 5 {
		h.Note = strings.Join(args[4:], " ")
	}
	s.AddHolding(h)
	fmt.Fprintf(env.Stdout, "added %s\n", h.Ticker)
	return 0
}

func portfolioPrice(env Env, s *portfolio.Store, args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(env.Stderr, "portfolio price: usage: price <ticker> <last_price>")
		return 1
	}
	idx := findHolding(s, args[0])
	if idx < 0 {
		fmt.Fprintf(env.Stderr, "portfolio price: no holding for %q\n", args[0])
		return 1
	}
	price, err := strconv.ParseFloat(args[1], 64)
	if err != nil || price < 0 {
		fmt.Fprintln(env.Stderr, "portfolio price: last_price must be a non-negative number")
		return 1
	}
	s.SetLastPrice(idx, price)
	fmt.Fprintln(env.Stdout, "ok")
	return 0
}

func portfolioRefresh(env Env, s *portfolio.Store, args []string) int {
	if env.PriceFetcher == nil {
		env.PriceFetcher = portfolio.NewYahooFetcher()
	}
	holdings := s.Holdings()
	if len(holdings) == 0 {
		fmt.Fprintln(env.Stdout, "(no holdings)")
		return 0
	}

	targets := make([]int, 0, len(holdings))
	if len(args) == 0 {
		for i := range holdings {
			targets = append(targets, i)
		}
	} else {
		idx := findHolding(s, args[0])
		if idx < 0 {
			fmt.Fprintf(env.Stderr, "portfolio refresh: no holding for %q\n", args[0])
			return 1
		}
		targets = append(targets, idx)
	}

	failures := 0
	for _, i := range targets {
		ticker := holdings[i].Ticker
		price, err := env.PriceFetcher.Fetch(ticker)
		if err != nil {
			fmt.Fprintf(env.Stderr, "%s: %v\n", ticker, err)
			failures++
			continue
		}
		s.SetLastPrice(i, price)
		fmt.Fprintf(env.Stdout, "%s  %s\n", ticker, strconv.FormatFloat(price, 'f', -1, 64))
	}
	if failures > 0 {
		return 1
	}
	return 0
}

func portfolioRm(env Env, s *portfolio.Store, args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(env.Stderr, "portfolio rm: usage: rm <ticker>")
		return 1
	}
	idx := findHolding(s, args[0])
	if idx < 0 {
		fmt.Fprintf(env.Stderr, "portfolio rm: no holding for %q\n", args[0])
		return 1
	}
	s.DeleteHolding(idx)
	fmt.Fprintln(env.Stdout, "ok")
	return 0
}

func portfolioWatchAdd(env Env, s *portfolio.Store, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(env.Stderr, "portfolio watch-add: usage: watch-add <ticker> [note]")
		return 1
	}
	w := portfolio.WatchItem{Ticker: strings.ToUpper(args[0])}
	if len(args) >= 2 {
		w.Note = strings.Join(args[1:], " ")
	}
	s.AddWatch(w)
	fmt.Fprintf(env.Stdout, "added %s\n", w.Ticker)
	return 0
}

func portfolioWatchRm(env Env, s *portfolio.Store, args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(env.Stderr, "portfolio watch-rm: usage: watch-rm <ticker>")
		return 1
	}
	idx := findWatch(s, args[0])
	if idx < 0 {
		fmt.Fprintf(env.Stderr, "portfolio watch-rm: no watch entry for %q\n", args[0])
		return 1
	}
	s.DeleteWatch(idx)
	fmt.Fprintln(env.Stdout, "ok")
	return 0
}

func findHolding(s *portfolio.Store, ticker string) int {
	t := strings.ToUpper(ticker)
	for i, h := range s.Holdings() {
		if h.Ticker == t {
			return i
		}
	}
	return -1
}

func findWatch(s *portfolio.Store, ticker string) int {
	t := strings.ToUpper(ticker)
	for i, w := range s.Watchlist() {
		if w.Ticker == t {
			return i
		}
	}
	return -1
}

// ---------- wifi ----------

func runWifi(env Env, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "wifi: list | current | connect | forget")
		return 1
	}
	switch args[0] {
	case "list":
		return wifiList(env)
	case "current":
		return wifiCurrent(env)
	case "connect":
		return wifiConnect(env, args[1:])
	case "forget":
		return wifiForget(env, args[1:])
	default:
		fmt.Fprintf(env.Stderr, "wifi: unknown subcommand %q\n", args[0])
		return 1
	}
}

func wifiInterface(env Env) (string, error) {
	out, err := env.Runner("networksetup", "-listallhardwareports")
	if err != nil {
		return "", fmt.Errorf("detect interface: %w", err)
	}
	return wifi.DetectWiFiInterface(string(out))
}

func wifiList(env Env) int {
	iface, err := wifiInterface(env)
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	out, err := env.Runner("networksetup", "-listpreferredwirelessnetworks", iface)
	if err != nil {
		fmt.Fprintf(env.Stderr, "list preferred networks: %v\n", err)
		return 1
	}
	nets := wifi.ParsePreferredNetworks(string(out))
	if len(nets) == 0 {
		fmt.Fprintln(env.Stdout, "(no preferred networks)")
		return 0
	}
	for _, n := range nets {
		fmt.Fprintln(env.Stdout, n.SSID)
	}
	return 0
}

func wifiCurrent(env Env) int {
	iface, err := wifiInterface(env)
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	out, err := env.Runner("networksetup", "-getairportnetwork", iface)
	if err == nil {
		line := strings.TrimSpace(string(out))
		const prefix = "Current Wi-Fi Network: "
		if strings.HasPrefix(line, prefix) {
			fmt.Fprintln(env.Stdout, strings.TrimPrefix(line, prefix))
			return 0
		}
	}
	out, err = env.Runner("ipconfig", "getsummary", iface)
	if err != nil {
		fmt.Fprintln(env.Stdout, "(not connected)")
		return 0
	}
	connected, ssid := wifi.ParseIpconfigSummary(string(out))
	if !connected {
		fmt.Fprintln(env.Stdout, "(not connected)")
		return 0
	}
	fmt.Fprintln(env.Stdout, ssid)
	return 0
}

func wifiConnect(env Env, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(env.Stderr, "wifi connect: usage: connect <ssid> [password]")
		return 1
	}
	iface, err := wifiInterface(env)
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	ssid := args[0]
	var cmd []string
	if len(args) >= 2 {
		cmd = []string{"-setairportnetwork", iface, ssid, args[1]}
	} else {
		cmd = []string{"-setairportnetwork", iface, ssid}
	}
	if out, err := env.Runner("networksetup", cmd...); err != nil {
		fmt.Fprintf(env.Stderr, "connect: %v: %s\n", err, strings.TrimSpace(string(out)))
		return 1
	}
	fmt.Fprintf(env.Stdout, "connected to %s\n", ssid)
	return 0
}

func wifiForget(env Env, args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(env.Stderr, "wifi forget: usage: forget <ssid>")
		return 1
	}
	iface, err := wifiInterface(env)
	if err != nil {
		fmt.Fprintln(env.Stderr, err)
		return 1
	}
	if out, err := env.Runner("networksetup", "-removepreferredwirelessnetwork", iface, args[0]); err != nil {
		fmt.Fprintf(env.Stderr, "forget: %v: %s\n", err, strings.TrimSpace(string(out)))
		return 1
	}
	fmt.Fprintln(env.Stdout, "ok")
	return 0
}

// ---------- calendar ----------

func runCalendar(env Env, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "calendar: list [days]")
		return 1
	}
	switch args[0] {
	case "list":
		days := 7
		if len(args) >= 2 {
			d, err := strconv.Atoi(args[1])
			if err != nil || d < 1 {
				fmt.Fprintln(env.Stderr, "calendar list: days must be a positive integer")
				return 1
			}
			days = d
		}
		raw, err := calendar.FetchRaw(days)
		if err != nil {
			fmt.Fprintln(env.Stderr, err)
			return 1
		}
		events := calendar.ParseEvents(raw)
		if len(events) == 0 {
			fmt.Fprintln(env.Stdout, "(no upcoming events)")
			return 0
		}
		for _, e := range events {
			fmt.Fprintf(env.Stdout, "%s  %s – %s  %s",
				e.Start.Format("Mon Jan 02"),
				e.Start.Format("15:04"),
				e.End.Format("15:04"),
				e.Title,
			)
			if e.Location != "" {
				fmt.Fprintf(env.Stdout, "  @ %s", e.Location)
			}
			fmt.Fprintf(env.Stdout, "  [%s]\n", e.Calendar)
		}
		return 0
	}
	fmt.Fprintf(env.Stderr, "calendar: unknown subcommand %q\n", args[0])
	return 1
}
