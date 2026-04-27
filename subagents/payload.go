package main

type ClaudePayload struct {
	SessionID     string `json:"session_id"`
	Model         Model  `json:"model"`
	Workspace     Workspace `json:"workspace"`
	ContextWindow ContextWindow `json:"context_window"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Workspace struct {
	CurrentDir string `json:"current_dir"`
	ProjectDir string `json:"project_dir"`
}

type ContextWindow struct {
	UsedPercentage      float64 `json:"used_percentage"`
	RemainingPercentage float64 `json:"remaining_percentage"`
	ContextWindowSize   int     `json:"context_window_size"`
}
