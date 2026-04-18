package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestEnv(t *testing.T) (Env, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := Env{
		ConfigDir: dir,
		Stdout:    &stdout,
		Stderr:    &stderr,
	}
	return env, &stdout, &stderr, dir
}

func TestIsSubcommand(t *testing.T) {
	cases := map[string]bool{
		"todo":      true,
		"portfolio": true,
		"wifi":      true,
		"calendar":  true,
		"actions":   true,
		"":          false,
		"--help":    false,
		"foo":       false,
	}
	for in, want := range cases {
		if got := IsSubcommand(in); got != want {
			t.Errorf("IsSubcommand(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestListActions(t *testing.T) {
	env, stdout, _, _ := newTestEnv(t)
	if code := Run(env, []string{"actions"}); code != 0 {
		t.Fatalf("actions exit code = %d, want 0", code)
	}
	out := stdout.String()
	for _, want := range []string{"todo list", "portfolio add", "wifi connect", "calendar list"} {
		if !strings.Contains(out, want) {
			t.Errorf("actions output missing %q; got %q", want, out)
		}
	}
}

func TestTodoAddListDoneRm(t *testing.T) {
	env, stdout, stderr, dir := newTestEnv(t)

	// add
	if code := Run(env, []string{"todo", "add", "buy", "coffee"}); code != 0 {
		t.Fatalf("todo add exit = %d, stderr=%q", code, stderr.String())
	}
	first := stdout.String()
	if !strings.Contains(first, "buy coffee") {
		t.Fatalf("todo add output missing text: %q", first)
	}
	// extract the assigned ID from "added  <id>  <text>"
	fields := strings.Fields(first)
	if len(fields) < 3 {
		t.Fatalf("todo add output malformed: %q", first)
	}
	id := fields[1]

	// list --all
	stdout.Reset()
	if code := Run(env, []string{"todo", "list"}); code != 0 {
		t.Fatalf("todo list exit = %d", code)
	}
	if !strings.Contains(stdout.String(), id) || !strings.Contains(stdout.String(), "[ ]") {
		t.Fatalf("todo list missing new task: %q", stdout.String())
	}

	// done
	stdout.Reset()
	if code := Run(env, []string{"todo", "done", id}); code != 0 {
		t.Fatalf("todo done exit = %d, stderr=%q", code, stderr.String())
	}

	// list --done shows it, --pending hides it
	stdout.Reset()
	Run(env, []string{"todo", "list", "--done"})
	if !strings.Contains(stdout.String(), id) {
		t.Errorf("todo list --done missing completed task: %q", stdout.String())
	}
	stdout.Reset()
	Run(env, []string{"todo", "list", "--pending"})
	if strings.Contains(stdout.String(), id) {
		t.Errorf("todo list --pending should hide completed task: %q", stdout.String())
	}

	// rm with unknown id returns non-zero
	if code := Run(env, []string{"todo", "rm", "nope"}); code == 0 {
		t.Error("todo rm with unknown id should fail")
	}

	// rm succeeds
	if code := Run(env, []string{"todo", "rm", id}); code != 0 {
		t.Fatalf("todo rm exit = %d, stderr=%q", code, stderr.String())
	}

	// file exists on disk
	if _, err := os.Stat(filepath.Join(dir, "todos.json")); err != nil {
		t.Errorf("expected todos.json persisted: %v", err)
	}
}

func TestTodoAddRequiresText(t *testing.T) {
	env, _, stderr, _ := newTestEnv(t)
	if code := Run(env, []string{"todo", "add"}); code == 0 {
		t.Error("todo add without args should fail")
	}
	if !strings.Contains(stderr.String(), "missing text") {
		t.Errorf("todo add stderr = %q", stderr.String())
	}
}

func TestPortfolioAddListPriceRm(t *testing.T) {
	env, stdout, stderr, _ := newTestEnv(t)

	if code := Run(env, []string{"portfolio", "add", "aapl", "10", "150", "180", "core holding"}); code != 0 {
		t.Fatalf("portfolio add exit = %d, stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	if code := Run(env, []string{"portfolio", "list"}); code != 0 {
		t.Fatalf("portfolio list exit = %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "AAPL") || !strings.Contains(out, "core holding") {
		t.Errorf("portfolio list missing row: %q", out)
	}
	if !strings.Contains(out, "cost_basis=1500.00") {
		t.Errorf("portfolio list missing cost basis total: %q", out)
	}

	// update price
	if code := Run(env, []string{"portfolio", "price", "aapl", "200"}); code != 0 {
		t.Fatalf("portfolio price exit = %d, stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	Run(env, []string{"portfolio", "list"})
	if !strings.Contains(stdout.String(), "market_value=2000.00") {
		t.Errorf("portfolio list after price missing market value: %q", stdout.String())
	}

	// invalid shares rejected
	if code := Run(env, []string{"portfolio", "add", "bad", "-5", "100"}); code == 0 {
		t.Error("portfolio add with negative shares should fail")
	}

	// unknown ticker rm fails
	if code := Run(env, []string{"portfolio", "rm", "zzzz"}); code == 0 {
		t.Error("portfolio rm unknown ticker should fail")
	}

	// rm succeeds
	if code := Run(env, []string{"portfolio", "rm", "AAPL"}); code != 0 {
		t.Fatalf("portfolio rm exit = %d, stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	Run(env, []string{"portfolio", "list"})
	if !strings.Contains(stdout.String(), "(no holdings)") {
		t.Errorf("portfolio list should be empty after rm: %q", stdout.String())
	}
}

type stubFetcher struct {
	prices map[string]float64
	errs   map[string]error
	calls  []string
}

func (f *stubFetcher) Fetch(ticker string) (float64, error) {
	f.calls = append(f.calls, ticker)
	if err, ok := f.errs[ticker]; ok {
		return 0, err
	}
	return f.prices[ticker], nil
}

func TestPortfolioRefreshAll(t *testing.T) {
	env, stdout, stderr, _ := newTestEnv(t)
	if code := Run(env, []string{"portfolio", "add", "aapl", "10", "150"}); code != 0 {
		t.Fatalf("seed aapl: stderr=%q", stderr.String())
	}
	if code := Run(env, []string{"portfolio", "add", "msft", "5", "300"}); code != 0 {
		t.Fatalf("seed msft: stderr=%q", stderr.String())
	}

	env.PriceFetcher = &stubFetcher{prices: map[string]float64{"AAPL": 201.5, "MSFT": 410.25}}
	stdout.Reset()
	if code := Run(env, []string{"portfolio", "refresh"}); code != 0 {
		t.Fatalf("refresh exit = %d, stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "AAPL  201.5") || !strings.Contains(out, "MSFT  410.25") {
		t.Errorf("refresh output = %q", out)
	}

	stdout.Reset()
	Run(env, []string{"portfolio", "list"})
	if !strings.Contains(stdout.String(), "market_value=4066.25") {
		t.Errorf("market value after refresh wrong: %q", stdout.String())
	}
}

func TestPortfolioRefreshSingle(t *testing.T) {
	env, stdout, stderr, _ := newTestEnv(t)
	Run(env, []string{"portfolio", "add", "aapl", "10", "150"})
	Run(env, []string{"portfolio", "add", "msft", "5", "300"})

	fetcher := &stubFetcher{prices: map[string]float64{"AAPL": 201.5, "MSFT": 410.25}}
	env.PriceFetcher = fetcher
	stdout.Reset()
	if code := Run(env, []string{"portfolio", "refresh", "msft"}); code != 0 {
		t.Fatalf("refresh exit = %d, stderr=%q", code, stderr.String())
	}
	if len(fetcher.calls) != 1 || fetcher.calls[0] != "MSFT" {
		t.Errorf("expected single fetch of MSFT, got %v", fetcher.calls)
	}
	if !strings.Contains(stdout.String(), "MSFT  410.25") {
		t.Errorf("refresh single output = %q", stdout.String())
	}
}

func TestPortfolioRefreshUnknownTicker(t *testing.T) {
	env, _, stderr, _ := newTestEnv(t)
	Run(env, []string{"portfolio", "add", "aapl", "10", "150"})
	env.PriceFetcher = &stubFetcher{}
	if code := Run(env, []string{"portfolio", "refresh", "zzzz"}); code == 0 {
		t.Error("refresh with unknown holding should fail")
	}
	if !strings.Contains(stderr.String(), "no holding for") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestPortfolioRefreshFetchErrorExitsNonZero(t *testing.T) {
	env, stdout, stderr, _ := newTestEnv(t)
	Run(env, []string{"portfolio", "add", "aapl", "10", "150"})
	Run(env, []string{"portfolio", "add", "msft", "5", "300"})

	env.PriceFetcher = &stubFetcher{
		prices: map[string]float64{"AAPL": 201.5},
		errs:   map[string]error{"MSFT": fmt.Errorf("boom")},
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(env, []string{"portfolio", "refresh"}); code != 1 {
		t.Errorf("refresh with partial failure exit = %d, want 1", code)
	}
	// Succeeded rows still written.
	if !strings.Contains(stdout.String(), "AAPL  201.5") {
		t.Errorf("expected AAPL update even when MSFT failed: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "MSFT: boom") {
		t.Errorf("expected MSFT error on stderr: %q", stderr.String())
	}
}

func TestPortfolioRefreshEmpty(t *testing.T) {
	env, stdout, _, _ := newTestEnv(t)
	env.PriceFetcher = &stubFetcher{}
	if code := Run(env, []string{"portfolio", "refresh"}); code != 0 {
		t.Errorf("refresh on empty store exit = %d", code)
	}
	if !strings.Contains(stdout.String(), "(no holdings)") {
		t.Errorf("stdout = %q", stdout.String())
	}
}

func TestPortfolioWatchRoundTrip(t *testing.T) {
	env, stdout, stderr, _ := newTestEnv(t)

	if code := Run(env, []string{"portfolio", "watch-add", "nvda", "earnings", "soon"}); code != 0 {
		t.Fatalf("watch-add exit = %d, stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	Run(env, []string{"portfolio", "watch"})
	if !strings.Contains(stdout.String(), "NVDA") || !strings.Contains(stdout.String(), "earnings soon") {
		t.Errorf("watch list missing entry: %q", stdout.String())
	}

	if code := Run(env, []string{"portfolio", "watch-rm", "nvda"}); code != 0 {
		t.Fatalf("watch-rm exit = %d", code)
	}
	stdout.Reset()
	Run(env, []string{"portfolio", "watch"})
	if !strings.Contains(stdout.String(), "(no watchlist entries)") {
		t.Errorf("watch should be empty: %q", stdout.String())
	}
}

func TestWifiListWithStubRunner(t *testing.T) {
	env, stdout, _, _ := newTestEnv(t)
	env.Runner = func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "networksetup" && len(args) > 0 && args[0] == "-listallhardwareports":
			return []byte("Hardware Port: Wi-Fi\nDevice: en0\nEthernet Address: aa:bb:cc:dd:ee:ff\n"), nil
		case name == "networksetup" && len(args) > 0 && args[0] == "-listpreferredwirelessnetworks":
			if args[1] != "en0" {
				return nil, fmt.Errorf("expected en0, got %s", args[1])
			}
			return []byte("Preferred networks on en0:\nHomeNet\nCafeWiFi\n"), nil
		}
		return nil, fmt.Errorf("unexpected call: %s %v", name, args)
	}
	if code := Run(env, []string{"wifi", "list"}); code != 0 {
		t.Fatalf("wifi list exit = %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "HomeNet") || !strings.Contains(out, "CafeWiFi") {
		t.Errorf("wifi list output = %q", out)
	}
}

func TestWifiCurrentWithStubRunner(t *testing.T) {
	env, stdout, _, _ := newTestEnv(t)
	env.Runner = func(name string, args ...string) ([]byte, error) {
		switch {
		case name == "networksetup" && args[0] == "-listallhardwareports":
			return []byte("Hardware Port: Wi-Fi\nDevice: en0\n"), nil
		case name == "networksetup" && args[0] == "-getairportnetwork":
			return []byte("Current Wi-Fi Network: HomeNet\n"), nil
		}
		return nil, fmt.Errorf("unexpected call: %s %v", name, args)
	}
	if code := Run(env, []string{"wifi", "current"}); code != 0 {
		t.Fatalf("wifi current exit = %d", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != "HomeNet" {
		t.Errorf("wifi current output = %q", got)
	}
}

func TestWifiConnectPassesPassword(t *testing.T) {
	env, _, _, _ := newTestEnv(t)
	var captured []string
	env.Runner = func(name string, args ...string) ([]byte, error) {
		if args[0] == "-listallhardwareports" {
			return []byte("Hardware Port: Wi-Fi\nDevice: en0\n"), nil
		}
		captured = append([]string{name}, args...)
		return nil, nil
	}
	if code := Run(env, []string{"wifi", "connect", "CafeWiFi", "hunter2"}); code != 0 {
		t.Fatalf("wifi connect exit = %d", code)
	}
	want := []string{"networksetup", "-setairportnetwork", "en0", "CafeWiFi", "hunter2"}
	if strings.Join(captured, " ") != strings.Join(want, " ") {
		t.Errorf("wifi connect called %v, want %v", captured, want)
	}
}

func TestUnknownSubcommandUsage(t *testing.T) {
	env, _, stderr, _ := newTestEnv(t)
	if code := Run(env, []string{"nope"}); code == 0 {
		t.Error("unknown subcommand should return non-zero")
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Errorf("expected usage output, got %q", stderr.String())
	}
}
