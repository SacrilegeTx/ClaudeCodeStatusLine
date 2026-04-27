package main

import (
	"strings"
	"testing"
)

func TestHookStartCreatesEntry(t *testing.T) {
	withFakeHome(t)

	payload := `{
		"session_id": "sess-1",
		"tool_use_id": "toolu_abc",
		"hook_event_name": "PreToolUse",
		"tool_name": "Task",
		"tool_input": {
			"subagent_type": "sdd-apply",
			"description": "implement",
			"prompt": "..."
		}
	}`

	if err := runHookStart(strings.NewReader(payload)); err != nil {
		t.Fatalf("runHookStart: %v", err)
	}

	s, _ := loadState()
	if len(s.Subagents) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(s.Subagents))
	}
	got := s.Subagents[0]
	if got.SessionID != "sess-1" || got.ID != "toolu_abc" || got.Name != "sdd-apply" {
		t.Errorf("unexpected entry: %+v", got)
	}
	if got.Status != StatusRunning {
		t.Errorf("status = %q, want running", got.Status)
	}
}

func TestHookStopMarksDone(t *testing.T) {
	withFakeHome(t)

	startPayload := `{"session_id":"s","tool_use_id":"t1","tool_input":{"subagent_type":"sdd-verify"}}`
	if err := runHookStart(strings.NewReader(startPayload)); err != nil {
		t.Fatalf("hook-start: %v", err)
	}

	stopPayload := `{
		"session_id": "s",
		"tool_use_id": "t1",
		"hook_event_name": "PostToolUse",
		"tool_name": "Task",
		"tool_response": {"is_error": false, "content": [{"type":"text","text":"..."}]}
	}`
	if err := runHookStop(strings.NewReader(stopPayload)); err != nil {
		t.Fatalf("hook-stop: %v", err)
	}

	s, _ := loadState()
	if s.Subagents[0].Status != StatusDone {
		t.Errorf("status = %q, want done", s.Subagents[0].Status)
	}
	if s.Subagents[0].EndedAt == nil {
		t.Error("EndedAt should be set")
	}
}

func TestHookStopMarksFailedOnError(t *testing.T) {
	withFakeHome(t)

	startPayload := `{"session_id":"s","tool_use_id":"t1","tool_input":{"subagent_type":"sdd-verify"}}`
	if err := runHookStart(strings.NewReader(startPayload)); err != nil {
		t.Fatalf("hook-start: %v", err)
	}

	stopPayload := `{"session_id":"s","tool_use_id":"t1","tool_response":{"is_error":true}}`
	if err := runHookStop(strings.NewReader(stopPayload)); err != nil {
		t.Fatalf("hook-stop: %v", err)
	}

	s, _ := loadState()
	if s.Subagents[0].Status != StatusFailed {
		t.Errorf("status = %q, want failed", s.Subagents[0].Status)
	}
}

func TestHookStartRejectsMissingFields(t *testing.T) {
	withFakeHome(t)

	cases := []string{
		`{"tool_use_id":"t","tool_input":{"subagent_type":"x"}}`,                        // no session_id
		`{"session_id":"s","tool_input":{"subagent_type":"x"}}`,                         // no tool_use_id
		`{"session_id":"s","tool_use_id":"t"}`,                                          // no subagent_type
		`{"session_id":"s","tool_use_id":"t","tool_input":{"subagent_type":""}}`,        // empty subagent_type
	}
	for i, p := range cases {
		if err := runHookStart(strings.NewReader(p)); err == nil {
			t.Errorf("case %d: expected error for %s", i, p)
		}
	}
}

func TestHookStopRejectsBadJSON(t *testing.T) {
	withFakeHome(t)

	if err := runHookStop(strings.NewReader("not json")); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
