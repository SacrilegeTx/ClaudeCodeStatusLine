package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func withFakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestLoadMissing(t *testing.T) {
	withFakeHome(t)

	got, err := loadState()
	if err != nil {
		t.Fatalf("loadState on missing file: %v", err)
	}
	if got == nil {
		t.Fatal("loadState returned nil state")
	}
	if len(got.Subagents) != 0 {
		t.Errorf("expected empty subagents, got %d", len(got.Subagents))
	}
}

func TestLoadEmpty(t *testing.T) {
	withFakeHome(t)

	path, err := stateFilePath()
	if err != nil {
		t.Fatalf("stateFilePath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write empty: %v", err)
	}

	got, err := loadState()
	if err != nil {
		t.Fatalf("loadState on empty file: %v", err)
	}
	if len(got.Subagents) != 0 {
		t.Errorf("expected empty subagents, got %d", len(got.Subagents))
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	withFakeHome(t)

	end := time.Date(2026, 4, 26, 22, 0, 0, 0, time.UTC)
	want := &State{Subagents: []Subagent{
		{
			ID:        "a1",
			SessionID: "sess-A",
			Name:      "explorer",
			Status:    StatusRunning,
			StartedAt: time.Date(2026, 4, 26, 21, 59, 0, 0, time.UTC),
		},
		{
			ID:        "a2",
			SessionID: "sess-B",
			Name:      "verifier",
			Status:    StatusDone,
			StartedAt: time.Date(2026, 4, 26, 21, 58, 0, 0, time.UTC),
			EndedAt:   &end,
			Tokens:    1234,
		},
	}}

	if err := saveState(want); err != nil {
		t.Fatalf("saveState: %v", err)
	}
	got, err := loadState()
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if len(got.Subagents) != len(want.Subagents) {
		t.Fatalf("len mismatch: got %d, want %d", len(got.Subagents), len(want.Subagents))
	}
	for i := range want.Subagents {
		w, g := want.Subagents[i], got.Subagents[i]
		if w.ID != g.ID || w.SessionID != g.SessionID || w.Name != g.Name || w.Status != g.Status || w.Tokens != g.Tokens {
			t.Errorf("entry %d: got %+v want %+v", i, g, w)
		}
		if !w.StartedAt.Equal(g.StartedAt) {
			t.Errorf("entry %d: StartedAt got %v want %v", i, g.StartedAt, w.StartedAt)
		}
		if (w.EndedAt == nil) != (g.EndedAt == nil) {
			t.Errorf("entry %d: EndedAt nilness mismatch", i)
		}
	}
}

func TestSaveLeavesNoTmp(t *testing.T) {
	withFakeHome(t)

	if err := saveState(&State{Subagents: []Subagent{{ID: "x", Name: "n", Status: StatusRunning, StartedAt: time.Now()}}}); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	dir, err := stateDir()
	if err != nil {
		t.Fatalf("stateDir: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("stale tmp file lingering: %s", e.Name())
		}
	}
}

// TestWithStateLockConcurrent is the contract test for the locking layer.
// N goroutines each do read-modify-write under withStateLock, appending a
// uniquely-named Subagent. If the lock works, every append survives and we
// see exactly N entries with the expected IDs. Without the lock, the
// read-modify-write race drops updates and the final count is < N.
func TestWithStateLockConcurrent(t *testing.T) {
	withFakeHome(t)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			err := withStateLock(func(s *State) error {
				s.Subagents = append(s.Subagents, Subagent{
					ID:        fmt.Sprintf("sa-%02d", i),
					Name:      "n",
					Status:    StatusRunning,
					StartedAt: time.Now(),
				})
				return nil
			})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("withStateLock returned error: %v", err)
	}

	got, err := loadState()
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if len(got.Subagents) != n {
		t.Fatalf("lost updates: got %d entries, want %d", len(got.Subagents), n)
	}

	ids := make([]string, len(got.Subagents))
	for i, s := range got.Subagents {
		ids[i] = s.ID
	}
	sort.Strings(ids)
	for i := 0; i < n; i++ {
		want := fmt.Sprintf("sa-%02d", i)
		if ids[i] != want {
			t.Errorf("missing/duplicate id at sorted index %d: got %q, want %q", i, ids[i], want)
		}
	}
}
