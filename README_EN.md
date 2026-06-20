# Spank-2

[**中文 README**](./README.md)

Simulate keystrokes or text input by tapping your MacBook.

Uses the Apple Silicon accelerometer to detect taps/slaps and simulates keyboard events via CoreGraphics.

## Requirements

- **Apple Silicon Mac** (M1+)
- **sudo privileges** (accelerometer requires direct IOKit access)
- **Accessibility permission**: System Settings > Privacy & Security > Accessibility > allow Terminal

# Project Structure

- **source-file**
  - **go.mod**: Go module definition
  - **go.sum**: Dependency checksums
  - **main.go**: Main program
  - **input_darwin.go**: macOS low-level input simulation
- **README.MD**: This document

## Commands

| Command | Description |
|---------|-------------|
| `sudo spank` | Default mode, tap to press Enter |
| `sudo spank --help` | Show help |

### --key / -k

Specify the key to press on tap.

```bash
sudo spank --key space          # Press space
sudo spank --key escape         # Press Escape
sudo spank --key enter          # Press Enter (default)
sudo spank --key a              # Press A
sudo spank --key tab            # Press Tab
sudo spank --key backspace      # Press Backspace
sudo spank --key up             # Press Up arrow
sudo spank --key down           # Press Down arrow
sudo spank --key left           # Press Left arrow
sudo spank --key right          # Press Right arrow
sudo spank --key 0-9            # Press number key
```

### --mouse / -m

Simulate mouse clicks.

```bash
sudo spank --mouse 0            # Tap for left click
sudo spank --mouse 1            # Tap for right click
```

### --command / -c

Execute a terminal command on tap. Supports `s=` segment syntax for cycling through different commands.

```bash
sudo spank -c 'open ~/.zshrc'                    # Tap to open .zshrc
sudo spank -c 's=| open ~/.zshrc|quit TextEdit'  # Cycle: open → quit
```

Commands run as the original user and load `~/.zshrc`, so custom aliases/functions are available. Command output is not shown.

### -v (Sensitivity)

| Value | Description |
|-------|-------------|
| `high` | A desk bump or light touch triggers |
| `mid` | Suitable for most scenarios (default) |
| `low` | Needs a firm slap to trigger |

```bash
sudo spank -v high              # High sensitivity
sudo spank -v low --mouse 0     # Low sensitivity + left click
```

### --word (Text Mode)

`--word` changes how `--key` behaves:

- **Single character** → simulated directly as a key press
- **Multiple characters** → copied to clipboard and pasted via `Cmd+V`
- **Segment output** → prefix with `s=` to define segments, cycling through on each tap

```bash
sudo spank --word --key hello          # Tap to type "hello"
sudo spank --word --key "你好世界"       # Tap to type Chinese characters
sudo spank --word --key a               # Single char = direct keypress
sudo spank --word --key 's=."Hi.I'm.Claude.'  # Cycle: Hi → I'm → Claude
```

#### Escape Sequences

| Sequence | Effect |
|----------|--------|
| `\n` | Newline |
| `\t` | Tab |
| `\\` | Literal backslash |

```bash
sudo spank --word --key "line1\nline2"    # Tap to type two lines
sudo spank --word --key "a\tb\tc"         # Tap to type tab-separated
```

#### Raw Strings

Wrap with `r"..."` to disable escape processing and input as-is.

```bash
sudo spank --word --key 'r"hello\n"'      # Types "hello\n" literally
```

## Combined Examples

### Daily Use

```bash
# Quick chat shortcut: tap to send a message
sudo spank -v high --word --key "/whisper\n"

# Paste a common command
sudo spank --word --key "sudo systemctl restart nginx\n"

# Tap to type an email address
sudo spank --word --key "example@gmail.com"

# Formatted input
sudo spank --word --key "Name:\tJohn\nAge:\t30\n"

# Low sensitivity + arrow keys for page turning
sudo spank -v low --key space

# Tap Tab to switch focus
sudo spank --key tab
```

### Segment Cycling

```bash
# Cycle through chat responses
sudo spank -w --key 's=."Okay ."Got it ."Be right there .'

# Type commands line by line
sudo spank -w --key 's=,git add .\n,git commit -m "update"\n,git push\n'

# Phone script cycling
sudo spank -w --key 's=|Hello, how can I help?|Thank you for calling, goodbye!|'

# Step-by-step guide
sudo spank -w --key 's=.Step 1: Open Settings.Step 2: Click Account.Step 3: Sign out.'
```

### Combinations

```bash
# High sensitivity + newline text
sudo spank -v high --word --key "docker ps\n"

# Low sensitivity + segment output (needs firm slap to advance)
sudo spank -v low -w --key 's=.nextpage.'

# Left click + low sensitivity
sudo spank -v low --mouse 0
```

### Practical Scenarios

```bash
# Quick replies
sudo spank -w --k 's=,Be right there!,Thanks!,Will reply later,'

# Code snippets
sudo spank -w --k 's=.import os.\nimport sys.\nimport json'

# Paste API key
sudo spank --word --k "sk-xxxxxxxxxxxxxxxx"

# Multi-line address input
sudo spank --word --k "123 Main St\nApt 4B\nNew York, NY\n"

# Markdown text with escapes
sudo spank -w --k "# Title\n\nThis is **bold** text\n- Item 1\n- Item 2\n"
```

## Build

```bash
cd source-file
GOFLAGS=-mod=mod go build -o ../spank -ldflags="-s -w" .
```

## Technical Details

- Reads the `AppleSPUHIDDevice` accelerometer directly via IOKit (using `taigrr/apple-silicon-accelerometer`)
- Simulates keyboard/mouse events using CoreGraphics CGEvent
- Paste mode: `pbcopy` + `Cmd+V` (supports Unicode characters including CJK)
- Zero external dependencies, single-file binary after compilation

Changes were based on the [Spank project](https://github.com/taigrr/spank).
