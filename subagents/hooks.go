package main

import (
	"encoding/json"
	"fmt"
	"io"
)

type hookStartPayload struct {
	SessionID string `json:"session_id"`
	ToolUseID string `json:"tool_use_id"`
	ToolInput struct {
		SubagentType string `json:"subagent_type"`
	} `json:"tool_input"`
}

type hookStopPayload struct {
	SessionID    string `json:"session_id"`
	ToolUseID    string `json:"tool_use_id"`
	ToolResponse struct {
		IsError bool `json:"is_error"`
	} `json:"tool_response"`
}

func runHookStart(stdin io.Reader) error {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("hook-start: read stdin: %w", err)
	}
	var p hookStartPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("hook-start: parse stdin: %w", err)
	}
	if p.SessionID == "" || p.ToolUseID == "" || p.ToolInput.SubagentType == "" {
		return fmt.Errorf("hook-start: missing session_id, tool_use_id or tool_input.subagent_type")
	}
	return runTrackStart([]string{p.SessionID, p.ToolUseID, p.ToolInput.SubagentType})
}

func runHookStop(stdin io.Reader) error {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("hook-stop: read stdin: %w", err)
	}
	var p hookStopPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("hook-stop: parse stdin: %w", err)
	}
	if p.SessionID == "" || p.ToolUseID == "" {
		return fmt.Errorf("hook-stop: missing session_id or tool_use_id")
	}
	status := "done"
	if p.ToolResponse.IsError {
		status = "failed"
	}
	return runTrackStop([]string{p.SessionID, p.ToolUseID, status})
}
