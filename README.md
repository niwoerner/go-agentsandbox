# go-agentsandbox

Lightweight filesystem sandbox for AI agents running on local machines.

Inspired by [sandbox-runtime](https://github.com/anthropic-experimental/sandbox-runtime). Created as a pure Go package for easy integration, with filesystem-only restrictions (network unrestricted by design).

**Primary use case:** Prevent agents from destructive filesystem operations on the host machine.

Prevents agents from accidentally:
- Deleting files outside the workspace (`rm -rf /`)
- Modifying sensitive directories (`~/.ssh`, `~/.aws`)
- Writing to system paths

## Features

- **Cross-platform**: macOS (sandbox-exec) and Linux (bubblewrap)
- **Filesystem-only**: Network unrestricted by design
- **Dual interface**: CLI tool + importable Go package

## Install

```bash
go install github.com/niwoerner/go-agentsandbox/cmd/agentsandbox@latest
```

Linux requires bubblewrap:
```bash
# Debian/Ubuntu
apt install bubblewrap

# Fedora/RHEL
dnf install bubblewrap
```

## Quick Start

```bash
# CLI
agentsandbox exec -- npm install

# Go package
sb, _ := sandbox.New(sandbox.DefaultConfig())
output, exitCode, _ := sb.Run(ctx, "npm install")
```

See [EXAMPLES.md](EXAMPLES.md) for detailed usage.

## How It Works

| Platform | Backend | Mechanism |
|----------|---------|-----------|
| macOS | sandbox-exec | Seatbelt syscall filtering |
| Linux | bubblewrap | Namespace isolation |

Both backends enforce filesystem restrictions at the kernel level. Even if a script tries to bypass restrictions, the actual syscalls are blocked.

### Configuration

Configuration is loaded from `~/.agent/sandbox/config.json` if it exists.

```json
{
  "allowWrite": ["/tmp", "."],
  "denyRead": ["~/.ssh", "~/.aws", "~/.gnupg"],
  "cleanEnv": false,
  "envAllowlist": [],
  "envDenylist": ["AWS_SECRET_ACCESS_KEY"]
}
```

**Priority (lowest to highest):**
1. Hardcoded defaults
2. Config file (`~/.agent/sandbox/config.json`)
3. CLI flags / SDK struct values

**Wildcards:** Use `"*"` for everything, e.g., `"allowWrite": ["*"]` allows all writes.

**Empty/omitted fields:** Use hardcoded defaults.

CLI flags:
```bash
agentsandbox exec --config ./custom.json -- npm install
agentsandbox exec --no-config -- npm install  # skip config file
```

### Default Values

**Writable paths (`allowWrite`):**
- Current working directory
- `/tmp`

**Protected paths (`denyRead`):**
- `~/.ssh`
- `~/.aws`
- `~/.gnupg`
- `~/.kube`
- `~/.docker`
- `~/.config/gh`

**Other defaults:**
- `cleanEnv`: false (pass through full environment)
- `envDenylist`: empty (configure as needed)
- Network: Unrestricted (by design)

### Alternative

For more advanced sandboxing (network restrictions, etc.), see [sandbox-runtime](https://github.com/anthropic-experimental/sandbox-runtime).

## Requirements

| Platform | Dependency | Notes |
|----------|------------|-------|
| macOS | None | sandbox-exec is built-in |
| Linux | bubblewrap | Install via package manager |

## Development

```bash
# Unit tests (run anywhere)
make test-unit

# Integration tests (current platform)
make test-integration

# Linux integration tests (via Docker)
make test-linux
```