# Claude Code Status Line

A custom status line for [Claude Code](https://claude.com/claude-code) that displays model info, token usage, rate limits, and reset times in a single compact line.

## Screenshot

![Status Line Screenshot](screenshot.png)

## What it shows

| Segment | Description |
|---------|-------------|
| **Model** | Current model name (e.g., Opus 4.6) |
| **Tokens** | Used / total context window tokens |
| **% Used / Remain** | Context window usage percentage |
| **Thinking** | Whether extended thinking is enabled |
| **5h** | 5-hour rate limit usage with progress bar and reset time |
| **7d** | 7-day rate limit usage with progress bar and reset time |
| **Extra** | Extra usage credits spent / limit (if enabled) |

Progress bars change color based on usage: green → orange → yellow → red.

## Requirements

- `jq` — for JSON parsing
- `curl` — for fetching usage data from the Anthropic API
- Claude Code with OAuth authentication (Pro/Max subscription)

## Installation

### Quick setup (recommended)

Copy the contents of `statusline.sh` and paste it into Claude Code with the prompt:

> Use this script as my status bar

Claude Code will save the script and configure `settings.json` for you automatically.

### Manual setup

1. Copy the script to your Claude config directory:

   ```bash
   cp statusline.sh ~/.claude/statusline.sh
   chmod +x ~/.claude/statusline.sh
   ```

2. Add the status line config to `~/.claude/settings.json`:

   ```json
   {
     "statusLine": {
       "type": "command",
       "command": "~/.claude/statusline.sh"
     }
   }
   ```

3. Restart Claude Code.

## Cross-platform support

The script works on both macOS and Linux:
- **macOS**: Uses Keychain for OAuth token retrieval and BSD `date` for time formatting
- **Linux**: Supports credentials file and GNOME Keyring, uses GNU `date`

## Caching

Usage data from the Anthropic API is cached for 60 seconds at `/tmp/claude/statusline-usage-cache.json` to avoid excessive API calls.

## License

MIT
