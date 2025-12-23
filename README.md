# go-agentsandbox

Lightweight filesystem sandbox for AI agents running on local machines.

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

### Default Configuration

**Writable paths (`AllowWrite`):**
- Current working directory
- `/tmp`

**Protected paths (`DenyRead`):**
- `~/.ssh`
- `~/.aws`
- `~/.gnupg`
- `~/.kube`
- `~/.docker`
- `~/.config/gh`

**Other defaults:**
- `CleanEnv`: false (pass through full environment)
- `EnvDenylist`: empty (configure as needed)
- Network: Unrestricted (by design)

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

## License

MIT
