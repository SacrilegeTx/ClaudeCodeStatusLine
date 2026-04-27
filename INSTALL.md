# Installation

This document is the authoritative install guide. It is written to be executed step-by-step by Claude Code when a user asks to install or update this status line.

## 1. Detect the operating system

- **macOS or Linux** → use `statusline.sh`
- **Windows** (PowerShell, CMD, Git Bash, or WSL) → use `statusline.ps1`

## 2. Clone the repo

Clone to `~/.claude/statusline/` on Unix, or `%USERPROFILE%\.claude\statusline\` on Windows. If that directory already exists and is a git clone of this repo, run `git pull` in it instead of re-cloning.

**macOS / Linux**

```bash
git clone https://github.com/daniel3303/ClaudeCodeStatusLine ~/.claude/statusline
chmod +x ~/.claude/statusline/statusline.sh
```

**Windows (PowerShell)**

```powershell
git clone https://github.com/daniel3303/ClaudeCodeStatusLine "$env:USERPROFILE\.claude\statusline"
```

## 3. Configure `settings.json`

Add (or update) the `statusLine` key in `~/.claude/settings.json` (Unix) or `%USERPROFILE%\.claude\settings.json` (Windows). If the file already contains other keys, merge — do not overwrite.

**macOS / Linux**

```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline/statusline.sh"
  }
}
```

**Windows**

```json
{
  "statusLine": {
    "type": "command",
    "command": "pwsh -NoProfile -ExecutionPolicy Bypass -File ~/.claude/statusline/statusline.ps1"
  }
}
```

If PowerShell 7+ (`pwsh`) is not installed, fall back to Windows PowerShell 5.1:

```json
{
  "statusLine": {
    "type": "command",
    "command": "powershell -NoProfile -ExecutionPolicy Bypass -File ~/.claude/statusline/statusline.ps1"
  }
}
```

> `-ExecutionPolicy Bypass` is **process-scoped** — it does not change your machine's PowerShell policy. Without it, a default `Restricted` or `AllSigned` policy (common on locked-down corporate machines) silently rejects the unsigned script and Claude Code shows no status line with no error.
>
> `~` is expanded by Claude Code v2.1.47+ on both Unix and Windows. On older Claude Code versions, replace `~/.claude/statusline/statusline.ps1` with `%USERPROFILE%\.claude\statusline\statusline.ps1` (CMD / PowerShell) or `$USERPROFILE\.claude\statusline\statusline.ps1` (Git Bash / WSL) — `%VAR%` expands only in CMD/PowerShell, `$VAR` only in bash shells.

## 4. (Optional) Enable the Subagent Monitor

Adds extra rows below the main status line tracking running and recently-finished `Task` subagents (see the README for visuals and the SDD color palette).

### 4.1 Build the binary

Requires Go 1.24+ (`brew install go`, `apt install golang`, or download from go.dev).

```bash
cd ~/.claude/statusline/subagents
go build .
```

This produces `~/.claude/statusline/subagents/subagents` (or `subagents.exe` on Windows). The binary is gitignored — rebuild after each `git pull` if upstream changed any Go source.

For Windows users on a non-Windows host, cross-compile:

```bash
GOOS=windows GOARCH=amd64 go build -o subagents.exe .
```

### 4.2 Add hooks to `settings.json`

Merge into the same `settings.json` file from step 3:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Task",
        "hooks": [
          { "type": "command", "command": "~/.claude/statusline/subagents/subagents hook-start" }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Task",
        "hooks": [
          { "type": "command", "command": "~/.claude/statusline/subagents/subagents hook-stop" }
        ]
      }
    ]
  }
}
```

`PreToolUse:Task` fires when Claude Code dispatches a subagent; `PostToolUse:Task` fires when it returns. The binary parses each hook's JSON from stdin, captures `session_id` + `tool_use_id` + `subagent_type`, and writes to `~/.claude/state/subagents.json` under a file lock.

### 4.3 (Recommended) Set `refreshInterval` for the spinner

Add `refreshInterval` to the existing `statusLine` block so the Braille spinner animates while subagents run:

```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline/statusline.sh",
    "refreshInterval": 300
  }
}
```

Without this, Claude Code only re-renders the status line on user/tool events, so the spinner appears frozen during long-running sync subagents.

## 5. Restart Claude Code

The status line is loaded at startup. After saving `settings.json`, tell the user to restart Claude Code (or start a new session) for the change to take effect.

## Updating

Pull the latest release:

```bash
git -C ~/.claude/statusline pull
```

No `settings.json` changes are needed — the command path is stable across versions.

## Uninstalling

1. Remove the `statusLine` block from `settings.json`.
2. Delete the clone: `rm -rf ~/.claude/statusline` (or the Windows equivalent).

## Requirements

- Claude Code (Pro/Max subscription for rate-limit and extra-usage display)
- `git` in `PATH`
- macOS / Linux: `jq` and `curl`
- Windows: PowerShell 5.1+ (default on Windows 10/11)
- Go 1.24+ (only if you enable the optional Subagent Monitor in step 4)

If `jq` is missing on macOS/Linux, install it with the system package manager (`brew install jq`, `apt install jq`, etc.).
