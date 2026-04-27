package main

import (
	"strings"
	"testing"
	"time"
)

func TestTrackStartCreatesEntry(t *testing.T) {
	withFakeHome(t)

	before := time.Now().UTC().Add(-time.Second)
	if err := runTrackStart([]string{"sess-1", "agent-abc", "Explore"}); err != nil {
		t.Fatalf("track-start: %v", err)
	}
	after := time.Now().UTC().Add(time.Second)

	s, err := loadState()
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if len(s.Subagents) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(s.Subagents))
	}
	got := s.Subagents[0]
	if got.ID != "agent-abc" || got.SessionID != "sess-1" || got.Name != "Explore" {
		t.Errorf("unexpected entry: %+v", got)
	}
	if got.Status != StatusRunning {
		t.Errorf("status = %q, want running", got.Status)
	}
	if got.StartedAt.Before(before) || got.StartedAt.After(after) {
		t.Errorf("StartedAt %v not in [%v, %v]", got.StartedAt, before, after)
	}
	if got.EndedAt != nil {
		t.Errorf("EndedAt should be nil on running entry, got %v", got.EndedAt)
	}
}

func TestTrackStartIdempotentPreservesStartedAt(t *testing.T) {
	withFakeHome(t)

	if err := runTrackStart([]string{"sess-1", "agent-abc", "Explore"}); err != nil {
		t.Fatalf("first track-start: %v", err)
	}
	first, _ := loadState()
	originalStart := first.Subagents[0].StartedAt

	time.Sleep(5 * time.Millisecond)
	if err := runTrackStart([]string{"sess-1", "agent-abc", "Explore"}); err != nil {
		t.Fatalf("second track-start: %v", err)
	}

	s, _ := loadState()
	if len(s.Subagents) != 1 {
		t.Fatalf("expected 1 entry after re-start, got %d", len(s.Subagents))
	}
	if !s.Subagents[0].StartedAt.Equal(originalStart) {
		t.Errorf("StartedAt was reset: was %v, now %v", originalStart, s.Subagents[0].StartedAt)
	}
}

func TestTrackStopMarksDone(t *testing.T) {
	withFakeHome(t)

	if err := runTrackStart([]string{"sess-1", "agent-abc", "Explore"}); err != nil {
		t.Fatalf("track-start: %v", err)
	}
	if err := runTrackStop([]string{"sess-1", "agent-abc", "done"}); err != nil {
		t.Fatalf("track-stop: %v", err)
	}

	s, _ := loadState()
	if len(s.Subagents) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(s.Subagents))
	}
	got := s.Subagents[0]
	if got.Status != StatusDone {
		t.Errorf("status = %q, want done", got.Status)
	}
	if got.EndedAt == nil {
		t.Error("EndedAt should be set after stop")
	}
}

func TestTrackStopMarksFailed(t *testing.T) {
	withFakeHome(t)

	if err := runTrackStart([]string{"sess-1", "agent-abc", "Explore"}); err != nil {
		t.Fatalf("track-start: %v", err)
	}
	if err := runTrackStop([]string{"sess-1", "agent-abc", "failed"}); err != nil {
		t.Fatalf("track-stop: %v", err)
	}
	s, _ := loadState()
	if s.Subagents[0].Status != StatusFailed {
		t.Errorf("status = %q, want failed", s.Subagents[0].Status)
	}
}

func TestTrackStopUnknownIsNoop(t *testing.T) {
	withFakeHome(t)

	// No prior track-start. Stop should not error.
	if err := runTrackStop([]string{"sess-1", "ghost", "done"}); err != nil {
		t.Fatalf("track-stop on unknown should be no-op, got: %v", err)
	}
	s, _ := loadState()
	if len(s.Subagents) != 0 {
		t.Errorf("state should be empty, got %d entries", len(s.Subagents))
	}
}

func TestTrackStopInvalidStatus(t *testing.T) {
	withFakeHome(t)

	err := runTrackStop([]string{"sess-1", "agent-abc", "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("error message = %q, want contains 'invalid status'", err.Error())
	}
}

func TestTrackBadArgs(t *testing.T) {
	withFakeHome(t)

	tests := []struct {
		name string
		fn   func([]string) error
		args []string
	}{
		{"start too few", runTrackStart, []string{"sess-1", "agent-abc"}},
		{"stop too few", runTrackStop, []string{"sess-1", "agent-abc"}},
		{"start empty session", runTrackStart, []string{"", "agent-abc", "Explore"}},
		{"stop empty agent", runTrackStop, []string{"sess-1", "", "done"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(tt.args); err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func renderWithSession(t *testing.T, sessionID string) string {
	t.Helper()
	payload := `{"session_id":"` + sessionID + `"}`
	var buf strings.Builder
	if err := runRender(strings.NewReader(payload), &buf); err != nil {
		t.Fatalf("runRender: %v", err)
	}
	return buf.String()
}

func TestRenderEmptyStateNoOutput(t *testing.T) {
	withFakeHome(t)

	out := renderWithSession(t, "sess-1")
	if out != "" {
		t.Errorf("expected empty output for empty state, got %q", out)
	}
}

func TestRenderFiltersBySession(t *testing.T) {
	withFakeHome(t)

	now := time.Now().UTC()
	state := &State{Subagents: []Subagent{
		{ID: "a", SessionID: "mine", Name: "sdd-apply", Status: StatusRunning, StartedAt: now.Add(-5 * time.Second)},
		{ID: "b", SessionID: "other", Name: "sdd-verify", Status: StatusRunning, StartedAt: now.Add(-3 * time.Second)},
	}}
	if err := saveState(state); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	out := renderWithSession(t, "mine")
	if !strings.Contains(out, "sdd-apply") {
		t.Errorf("expected sdd-apply in output, got %q", out)
	}
	if strings.Contains(out, "sdd-verify") {
		t.Errorf("output leaked other session's entry: %q", out)
	}
}

func TestRenderHidesOldFinished(t *testing.T) {
	withFakeHome(t)

	now := time.Now().UTC()
	old := now.Add(-30 * time.Second)
	state := &State{Subagents: []Subagent{
		{ID: "a", SessionID: "s", Name: "sdd-apply", Status: StatusDone,
			StartedAt: now.Add(-60 * time.Second), EndedAt: &old},
	}}
	if err := saveState(state); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	out := renderWithSession(t, "s")
	if out != "" {
		t.Errorf("finished >10s ago should not render, got %q", out)
	}
}

func TestRenderShowsRecentFinished(t *testing.T) {
	withFakeHome(t)

	now := time.Now().UTC()
	recent := now.Add(-3 * time.Second)
	state := &State{Subagents: []Subagent{
		{ID: "a", SessionID: "s", Name: "sdd-verify", Status: StatusDone,
			StartedAt: now.Add(-15 * time.Second), EndedAt: &recent},
	}}
	if err := saveState(state); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	out := renderWithSession(t, "s")
	if !strings.Contains(out, "sdd-verify") {
		t.Errorf("expected sdd-verify in output, got %q", out)
	}
	if !strings.Contains(out, "done") {
		t.Errorf("expected 'done' status in output, got %q", out)
	}
}

func TestRenderSortedByStartedAt(t *testing.T) {
	withFakeHome(t)

	now := time.Now().UTC()
	state := &State{Subagents: []Subagent{
		{ID: "newer", SessionID: "s", Name: "sdd-tasks", Status: StatusRunning,
			StartedAt: now.Add(-1 * time.Second)},
		{ID: "older", SessionID: "s", Name: "sdd-explore", Status: StatusRunning,
			StartedAt: now.Add(-10 * time.Second)},
	}}
	if err := saveState(state); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	out := renderWithSession(t, "s")
	iOlder := strings.Index(out, "sdd-explore")
	iNewer := strings.Index(out, "sdd-tasks")
	if iOlder < 0 || iNewer < 0 {
		t.Fatalf("missing entries in output: %q", out)
	}
	if iOlder >= iNewer {
		t.Errorf("expected older entry first, got:\n%s", out)
	}
}

func TestTruncateName(t *testing.T) {
	cases := []struct {
		name string
		max  int
		want string
	}{
		{"short", 24, "short"},
		{"vercel:performance-optimizer", 24, "vercel:performance-opti…"},
		{"exact-twenty-four-chars!", 24, "exact-twenty-four-chars!"},
		{"", 24, ""},
		{"abc", 1, "abc"}, // max <= 1 → unchanged
	}
	for _, tc := range cases {
		got := truncateName(tc.name, tc.max)
		if got != tc.want {
			t.Errorf("truncateName(%q, %d) = %q, want %q", tc.name, tc.max, got, tc.want)
		}
		if len([]rune(got)) > tc.max && tc.max > 1 {
			t.Errorf("truncateName(%q, %d) = %q exceeds max width", tc.name, tc.max, got)
		}
	}
}

func TestGlyphPerStatus(t *testing.T) {
	now := time.Now()
	if g := glyph(StatusDone, now); g != "✓" {
		t.Errorf("done glyph = %q, want ✓", g)
	}
	if g := glyph(StatusFailed, now); g != "✗" {
		t.Errorf("failed glyph = %q, want ✗", g)
	}
	g := glyph(StatusRunning, now)
	found := false
	for _, f := range spinnerFrames {
		if f == g {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("running glyph %q is not a spinner frame", g)
	}
}

func TestSpinnerCyclesOverTime(t *testing.T) {
	base := time.Unix(0, 0)
	a := spinnerFrame(base)
	b := spinnerFrame(base.Add(500 * time.Millisecond))
	if a == b {
		t.Errorf("spinner did not advance after 500ms: still %q", a)
	}
}

func TestStyledDurationThresholds(t *testing.T) {
	cases := []struct {
		name      string
		status    SubagentStatus
		dur       time.Duration
		wantColor string // empty = expect dim, non-empty = expect this fg code
	}{
		{"running short → dim", StatusRunning, 5 * time.Second, ""},
		{"running >30s → warn yellow", StatusRunning, 45 * time.Second, "\x1b[38;5;179m"},
		{"running >2m → danger orange", StatusRunning, 3 * time.Minute, "\x1b[38;5;208m"},
		{"done → dim regardless of duration", StatusDone, 5 * time.Minute, ""},
		{"failed → dim regardless of duration", StatusFailed, 10 * time.Minute, ""},
	}
	for _, tc := range cases {
		got := styledDuration(tc.status, tc.dur)
		if tc.wantColor == "" {
			if !strings.Contains(got, "\x1b[2m") {
				t.Errorf("%s: expected dim, got %q", tc.name, got)
			}
		} else {
			if !strings.Contains(got, tc.wantColor) {
				t.Errorf("%s: expected color %q in %q", tc.name, tc.wantColor, got)
			}
		}
	}
}

func TestRenderTruncatesLongNames(t *testing.T) {
	withFakeHome(t)

	state := &State{Subagents: []Subagent{
		{ID: "a", SessionID: "s", Name: "vercel:performance-optimizer", Status: StatusRunning,
			StartedAt: time.Now().UTC().Add(-time.Second)},
	}}
	if err := saveState(state); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	out := renderWithSession(t, "s")
	if !strings.Contains(out, "…") {
		t.Errorf("expected ellipsis in truncated output, got %q", out)
	}
	if strings.Contains(out, "performance-optimizer") {
		t.Errorf("full long name should not appear, got %q", out)
	}
}

func TestRenderUsesPhaseColor(t *testing.T) {
	withFakeHome(t)

	state := &State{Subagents: []Subagent{
		{ID: "a", SessionID: "s", Name: "sdd-apply", Status: StatusRunning,
			StartedAt: time.Now().UTC().Add(-time.Second)},
	}}
	if err := saveState(state); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	out := renderWithSession(t, "s")
	want := "\x1b[38;5;71m" // sdd-apply → sage (256-color 71)
	if !strings.Contains(out, want) {
		t.Errorf("expected ANSI color %q for sdd-apply, got %q", want, out)
	}
}

// TestTrackIsolatesBySession is the contract test for Option A: two
// Claude Code sessions can each have a subagent with the same agent_id
// without interfering. Stopping one must not affect the other.
func TestTrackIsolatesBySession(t *testing.T) {
	withFakeHome(t)

	if err := runTrackStart([]string{"sess-A", "shared-id", "Explore"}); err != nil {
		t.Fatalf("start A: %v", err)
	}
	if err := runTrackStart([]string{"sess-B", "shared-id", "Plan"}); err != nil {
		t.Fatalf("start B: %v", err)
	}

	s, _ := loadState()
	if len(s.Subagents) != 2 {
		t.Fatalf("expected 2 separate entries, got %d", len(s.Subagents))
	}

	// Stop only session A's entry.
	if err := runTrackStop([]string{"sess-A", "shared-id", "done"}); err != nil {
		t.Fatalf("stop A: %v", err)
	}

	s, _ = loadState()
	var sessA, sessB *Subagent
	for i := range s.Subagents {
		switch s.Subagents[i].SessionID {
		case "sess-A":
			sessA = &s.Subagents[i]
		case "sess-B":
			sessB = &s.Subagents[i]
		}
	}
	if sessA == nil || sessB == nil {
		t.Fatal("missing session A or B entry")
	}
	if sessA.Status != StatusDone {
		t.Errorf("sess-A status = %q, want done", sessA.Status)
	}
	if sessB.Status != StatusRunning {
		t.Errorf("sess-B status = %q, want running (must not be touched)", sessB.Status)
	}
	if sessB.EndedAt != nil {
		t.Error("sess-B EndedAt should still be nil")
	}
}
