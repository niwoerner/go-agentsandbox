# Examples

## CLI

```bash
# Basic usage
agentsandbox exec -- npm install
agentsandbox exec --workdir /project -- make build

# Custom config file
agentsandbox exec --config ./sandbox.json -- ./script.sh
agentsandbox exec --no-config -- ls -la

# Override paths (replaces config)
agentsandbox exec --allow-write /var/cache -- apt-get update
agentsandbox exec --deny-read ~/.secrets -- ./build.sh

# Dry run
agentsandbox exec --dry-run -- rm -rf /
```

## Go Package

```go
// Default config (loads ~/.agent/sandbox/config.json if exists)
sb, _ := sandbox.New(sandbox.DefaultConfig())
output, exitCode, _ := sb.Run(ctx, "npm install")

// Custom config
sb, _ := sandbox.New(sandbox.Config{
    Workdir:     "/project",
    AllowWrite:  []string{"/project", "/tmp"},
    DenyRead:    []string{"~/.ssh", "~/.aws"},
    EnvDenylist: []string{"AWS_SECRET_ACCESS_KEY", "GITHUB_TOKEN"},
})

// With stdin
sb.RunWithStdin(ctx, "cat", strings.NewReader("hello"))

// With timeout
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
sb.Run(ctx, "npm install")
```

## Config File

`~/.agent/sandbox/config.json`:
```json
{
  "allowWrite": ["/tmp"],
  "denyRead": ["~/.ssh", "~/.aws", "~/.gnupg"],
  "envDenylist": ["AWS_SECRET_ACCESS_KEY"]
}
```

Use `"*"` wildcard: `"allowWrite": ["*"]` allows all writes.
