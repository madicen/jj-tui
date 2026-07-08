package model

// P5.2 auto-refresh guard tests. These lock shouldSilentReload: the single predicate that
// decides whether the heartbeat tick kicks off a silent background graph reload. The feature
// must be OFF by default and must NEVER fire while a modal is open or a jj command is in flight.

import (
	"testing"
	"time"

	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
)

func autoRefreshModel(t *testing.T, seconds int) *Model {
	t.Helper()
	m := newGoldenModel(t)
	m.appState.JJService = &jj.Service{RepoPath: "/test/repo"}
	cfg := &config.Config{}
	if seconds > 0 {
		cfg.AutoRefreshSeconds = &seconds
	}
	m.appState.Config = cfg
	m.SetViewMode(state.ViewCommitGraph)
	return m
}

func TestShouldSilentReloadOffByDefault(t *testing.T) {
	m := autoRefreshModel(t, 0)
	defer m.Close()
	if m.shouldSilentReload(time.Now()) {
		t.Fatal("auto-refresh must be OFF when ui.auto_refresh_seconds is 0 (default)")
	}
}

func TestShouldSilentReloadOnWhenIdle(t *testing.T) {
	m := autoRefreshModel(t, 5)
	defer m.Close()
	if !m.shouldSilentReload(time.Now()) {
		t.Fatal("auto-refresh should fire when enabled, idle, and no modal is open")
	}
}

func TestShouldSilentReloadSkipsWhileModalOpen(t *testing.T) {
	m := autoRefreshModel(t, 5)
	defer m.Close()
	// A form modal (Edit Description) is open.
	m.SetViewMode(state.ViewEditDescription)
	if m.shouldSilentReload(time.Now()) {
		t.Fatal("auto-refresh must skip while a modal is open")
	}
	// An error modal open over the graph must also suppress it.
	m.SetViewMode(state.ViewCommitGraph)
	m.applyEffect(effShowError{err: errBoom})
	if m.shouldSilentReload(time.Now()) {
		t.Fatal("auto-refresh must skip while the error modal is open")
	}
}

func TestShouldSilentReloadSkipsWhileInFlight(t *testing.T) {
	m := autoRefreshModel(t, 5)
	defer m.Close()
	m.silentReloadInFlight = true
	if m.shouldSilentReload(time.Now()) {
		t.Fatal("auto-refresh must skip while a silent reload is already in flight")
	}
	m.silentReloadInFlight = false
	m.appState.Loading = true
	if m.shouldSilentReload(time.Now()) {
		t.Fatal("auto-refresh must skip while a jj command is loading")
	}
}

func TestShouldSilentReloadHonorsInterval(t *testing.T) {
	m := autoRefreshModel(t, 30)
	defer m.Close()
	now := time.Now()
	m.lastAutoRefresh = now
	// Only 10s elapsed against a 30s interval: too soon.
	if m.shouldSilentReload(now.Add(10 * time.Second)) {
		t.Fatal("auto-refresh must respect the configured minimum spacing")
	}
	// 31s elapsed: due.
	if !m.shouldSilentReload(now.Add(31 * time.Second)) {
		t.Fatal("auto-refresh should fire once the configured interval has elapsed")
	}
}

// TestHandleTickMsgDispatchesReloadWhenEnabled proves the heartbeat tick actually kicks off a
// silent reload when auto-refresh is on and idle (flipping silentReloadInFlight), and that it
// does NOT when the feature is off or a modal is open. This is the tick-side counterpart to the
// SilentRepositoryLoadedMsg apply test which proves the reload result updates the graph.
func TestHandleTickMsgDispatchesReloadWhenEnabled(t *testing.T) {
	t.Run("on_and_idle_dispatches", func(t *testing.T) {
		m := autoRefreshModel(t, 1)
		defer m.Close()
		m.handleTickMsg(time.Now())
		if !m.silentReloadInFlight {
			t.Fatal("enabled+idle tick should dispatch a silent reload (silentReloadInFlight=true)")
		}
	})
	t.Run("off_does_not_dispatch", func(t *testing.T) {
		m := autoRefreshModel(t, 0)
		defer m.Close()
		m.handleTickMsg(time.Now())
		if m.silentReloadInFlight {
			t.Fatal("auto-refresh off: tick must not dispatch a silent reload")
		}
	})
	t.Run("modal_open_does_not_dispatch", func(t *testing.T) {
		m := autoRefreshModel(t, 1)
		defer m.Close()
		m.SetViewMode(state.ViewWorkspaces)
		m.handleTickMsg(time.Now())
		if m.silentReloadInFlight {
			t.Fatal("modal open: tick must not dispatch a silent reload")
		}
	})
}

var errBoom = errTest("boom")

type errTest string

func (e errTest) Error() string { return string(e) }
