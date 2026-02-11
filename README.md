# flow

A minimal task management TUI app built with Go, BubbleTea, and SQLite.

## Install

```bash
go install github.com/nissyi-gh/flow@latest
```

Or build from source:

```bash
git clone https://github.com/nissyi-gh/flow.git
cd flow
go build -o flow .
```

## Usage

```bash
flow
```

### Keybindings

| Key | Action |
|-----|--------|
| `a` / `n` | Add new task |
| `enter` / `x` | Toggle completion |
| `d` | Delete task (with confirmation) |
| `/` | Filter tasks |
| `j` / `k`, `↑` / `↓` | Navigate |
| `q` / `ctrl+c` | Quit |

## Data Storage

Tasks are stored in a SQLite database at `$XDG_DATA_HOME/flow/flow.db` (defaults to `~/.local/share/flow/flow.db`).
