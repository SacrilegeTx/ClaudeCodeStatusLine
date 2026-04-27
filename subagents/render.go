package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

const (
	recentFinishWindow = 10 * time.Second
	maxNameWidth       = 24
	warnAfter          = 30 * time.Second
	dangerAfter        = 2 * time.Minute
)

var phaseColors = map[string]int{
	"sdd-explore": 67,
	"sdd-propose": 96,
	"sdd-spec":    73,
	"sdd-design":  103,
	"sdd-tasks":   137,
	"sdd-apply":   71,
	"sdd-verify":  173,
	"sdd-archive": 102,
	"sdd-init":    66,
	"sdd-onboard": 138,
}

const (
	defaultColor = 244 // dim gray for non-SDD agents
	warnColor    = 179 // muted yellow for long-running (>30s)
	dangerColor  = 208 // orange for very long (>2m)
	failColor    = 167 // muted red for failed glyph
	doneColor    = 71  // sage for done glyph (overrides phase color for the ✓)
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func colorFor(name string) int {
	if c, ok := phaseColors[name]; ok {
		return c
	}
	return defaultColor
}

func fg(code int, s string) string {
	return fmt.Sprintf("\x1b[38;5;%dm%s\x1b[0m", code, s)
}

func dim(s string) string {
	return fmt.Sprintf("\x1b[2m%s\x1b[0m", s)
}

func spinnerFrame(now time.Time) string {
	idx := (now.UnixMilli() / 100) % int64(len(spinnerFrames))
	if idx < 0 {
		idx = 0
	}
	return spinnerFrames[idx]
}

func glyph(status SubagentStatus, now time.Time) string {
	switch status {
	case StatusRunning:
		return spinnerFrame(now)
	case StatusDone:
		return "✓"
	case StatusFailed:
		return "✗"
	default:
		return "●"
	}
}

func glyphColor(status SubagentStatus, phaseColor int) int {
	switch status {
	case StatusDone:
		return doneColor
	case StatusFailed:
		return failColor
	default:
		return phaseColor
	}
}

func truncateName(name string, max int) string {
	if max <= 1 {
		return name
	}
	runes := []rune(name)
	if len(runes) <= max {
		return name
	}
	return string(runes[:max-1]) + "…"
}

func styledDuration(status SubagentStatus, d time.Duration) string {
	s := formatDuration(d)
	if status != StatusRunning {
		return dim(s)
	}
	switch {
	case d >= dangerAfter:
		return fg(dangerColor, s)
	case d >= warnAfter:
		return fg(warnColor, s)
	default:
		return dim(s)
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

func runRender(stdin io.Reader, stdout io.Writer) error {
	var payload ClaudePayload
	if data, err := io.ReadAll(stdin); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &payload)
	}
	state, err := loadState()
	if err != nil {
		return nil
	}
	now := time.Now().UTC()
	var visible []Subagent
	for _, sa := range state.Subagents {
		if payload.SessionID != "" && sa.SessionID != payload.SessionID {
			continue
		}
		switch sa.Status {
		case StatusRunning:
			visible = append(visible, sa)
		case StatusDone, StatusFailed:
			if sa.EndedAt != nil && now.Sub(*sa.EndedAt) <= recentFinishWindow {
				visible = append(visible, sa)
			}
		}
	}
	if len(visible) == 0 {
		return nil
	}
	sort.Slice(visible, func(i, j int) bool {
		return visible[i].StartedAt.Before(visible[j].StartedAt)
	})
	for _, sa := range visible {
		c := colorFor(sa.Name)
		var dur time.Duration
		if sa.Status == StatusRunning {
			dur = now.Sub(sa.StartedAt)
		} else if sa.EndedAt != nil {
			dur = sa.EndedAt.Sub(sa.StartedAt)
		}
		fmt.Fprintf(stdout, "%s %s · %s · %s\n",
			fg(glyphColor(sa.Status, c), glyph(sa.Status, now)),
			fg(c, truncateName(sa.Name, maxNameWidth)),
			dim(string(sa.Status)),
			styledDuration(sa.Status, dur),
		)
	}
	return nil
}

func runTrackStart(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("track-start: expected <session-id> <agent-id> <agent-type>, got %d args", len(args))
	}
	sessionID, agentID, agentType := args[0], args[1], args[2]
	if sessionID == "" || agentID == "" || agentType == "" {
		return fmt.Errorf("track-start: session-id, agent-id, agent-type must be non-empty")
	}
	return withStateLock(func(s *State) error {
		for i, sa := range s.Subagents {
			if sa.SessionID == sessionID && sa.ID == agentID {
				s.Subagents[i].Name = agentType
				s.Subagents[i].Status = StatusRunning
				s.Subagents[i].EndedAt = nil
				return nil
			}
		}
		s.Subagents = append(s.Subagents, Subagent{
			ID:        agentID,
			SessionID: sessionID,
			Name:      agentType,
			Status:    StatusRunning,
			StartedAt: time.Now().UTC(),
		})
		return nil
	})
}

func runTrackStop(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("track-stop: expected <session-id> <agent-id> <status>, got %d args", len(args))
	}
	sessionID, agentID, statusStr := args[0], args[1], args[2]
	if sessionID == "" || agentID == "" {
		return fmt.Errorf("track-stop: session-id and agent-id must be non-empty")
	}
	var status SubagentStatus
	switch statusStr {
	case "done":
		status = StatusDone
	case "failed":
		status = StatusFailed
	default:
		return fmt.Errorf("track-stop: invalid status %q (want done|failed)", statusStr)
	}
	return withStateLock(func(s *State) error {
		now := time.Now().UTC()
		for i, sa := range s.Subagents {
			if sa.SessionID == sessionID && sa.ID == agentID {
				s.Subagents[i].Status = status
				s.Subagents[i].EndedAt = &now
				return nil
			}
		}
		fmt.Fprintf(os.Stderr, "track-stop: no subagent found for session=%s agent=%s (start hook missed?)\n", sessionID, agentID)
		return nil
	})
}
