# Claude Code Status Line

A custom status line for [Claude Code](https://claude.com/claude-code) that displays model info, token usage, rate limits, and reset times in a single compact line. It runs as an external shell command, so it does not slow down Claude Code or consume any extra tokens.

## Screenshot

![Status Line Screenshot](screenshot.png)

## What it shows

| Segment | Description |
|---------|-------------|
| **Model** | Current model name (e.g., Opus 4.7) |
| **CWD@Branch** | Current folder name, git branch, and file changes (+/-) |
| **Tokens** | Used / total context window tokens (% used) |
| **Effort** | Reasoning effort level (low, med, high, xhigh) |
| **5h** | 5-hour rate limit usage percentage and reset time |
| **7d** | 7-day rate limit usage percentage and reset time |
| **Extra** | Extra usage credits spent / limit (if enabled) |
| **Update** | Appears when a new version is available (checked every 24h) |

Usage percentages are color-coded: green (<50%) → yellow (≥50%) → orange (≥70%) → red (≥90%).

## Subagent Monitor (optional)

When enabled, additional rows render below the main status line tracking Claude Code's running and recently-finished `Task` subagents — particularly useful for SDD orchestrator workflows where multiple agents run in parallel.

```
Opus 4.7 1M | statusline@main | 110k/1m (11%) | effort: high
⠋ sdd-design · running · 2s
✓ sdd-apply · done · 5s
✗ sdd-verify · failed · 12s
```

| Visual | Meaning |
|--------|---------|
| Braille spinner (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`) | Subagent is running |
| `✓` | Finished successfully (visible 10s after completion) |
| `✗` | Failed (visible 10s after completion) |
| Yellow / orange duration | Running >30s (yellow) or >2m (orange) — flags stuck agents |
| `…` ellipsis | Subagent type name truncated at 24 chars |

### SDD phase colors

Each SDD phase has its own muted 256-color shade so the orchestrator's current step is identifiable at a glance:

| Phase | Color |
|-------|-------|
| `sdd-explore` | steel blue |
| `sdd-propose` | muted plum |
| `sdd-spec` | muted teal |
| `sdd-design` | slate |
| `sdd-tasks` | tan |
| `sdd-apply` | sage |
| `sdd-verify` | salmon |
| `sdd-archive` | mauve gray |
| `sdd-init` | muted cyan |
| `sdd-onboard` | rose |
| Other (`Explore`, `general-purpose`, `vercel:*`, …) | dim gray |

### How it works

A small Go binary in `subagents/` is invoked by the status line on each render. It tracks subagent lifecycle through Claude Code's `PreToolUse:Task` and `PostToolUse:Task` hooks, persisting state under `~/.claude/state/subagents.json` with file-based locking so multiple Claude Code instances stay isolated by `session_id`.

To enable, build the binary (`cd subagents && go build .`) and add the hooks shown in [INSTALL.md](INSTALL.md). Setting `refreshInterval: 300` on `statusLine` is recommended so the spinner actually animates.

## Installation

Ask Claude Code:

> Clone https://github.com/daniel3303/ClaudeCodeStatusLine to `~/.claude/statusline/` (or `%USERPROFILE%\.claude\statusline\` on Windows) and configure it as my status bar by following its INSTALL.md.

Claude will clone the repo to that path, pick the right script for your OS, and update `settings.json`. Full step-by-step instructions Claude follows live in [INSTALL.md](INSTALL.md).

Restart Claude Code after Claude saves the configuration.

### Updating

When the status line shows a new release is available, ask Claude:

> Find my installed status bar and update it.

Or update it yourself:

```bash
git -C ~/.claude/statusline pull
```

No `settings.json` changes are needed — the path stays valid across versions.

## Requirements

- Claude Code with OAuth authentication (Pro/Max subscription for rate-limit and extra-usage data)
- `git` in `PATH`
- macOS / Linux: `jq` and `curl`
- Windows: PowerShell 5.1+ (default on Windows 10/11)

## Caching

Usage data from the Anthropic API is cached for 60 seconds at `/tmp/claude/statusline-usage-cache-<hash>.json` (or `%TEMP%\claude\...` on Windows). Release checks are cached for 24 hours. Both caches are shared across concurrent Claude Code instances to avoid rate limits.

## Update Notifications

The status line checks GitHub for new releases once every 24 hours. When a newer version is available, a second line appears below the status line. The check fails silently if the API is unreachable.

## License

MIT

## Author

Daniel Oliveira

[![Website](https://img.shields.io/badge/Website-FF6B6B?style=for-the-badge&logo=safari&logoColor=white)](https://danielapoliveira.com/)
[![X](https://img.shields.io/badge/X-000000?style=for-the-badge&logo=x&logoColor=white)](https://x.com/daniel_not_nerd)
[![LinkedIn](https://img.shields.io/badge/LinkedIn-0077B5?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/in/daniel-ap-oliveira/)
