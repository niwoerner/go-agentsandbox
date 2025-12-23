# Examples

## CLI Usage

### Basic

```bash
# Run command in sandbox (writes allowed only in current dir + /tmp)
agentsandbox exec -- npm install
agentsandbox exec -- go build ./...

# Specify workspace
agentsandbox exec --workdir /path/to/project -- make build

# Dry run (show what would execute)
agentsandbox exec --dry-run -- rm -rf /
```

### Additional Options

```bash
# Allow writing to additional paths
agentsandbox exec --allow-write /var/cache -- apt-get update

# Protect additional paths from reading
agentsandbox exec --deny-read ~/.config/secrets -- ./script.sh

# Clean environment (only PATH, HOME, USER, TERM passed through)
agentsandbox exec --clean-env -- ./build.sh
```

## Go Package Usage

### Basic

```go
package main

import (
    "context"
    "log"

    "github.com/niwoerner/go-agentsandbox/sandbox"
)

func main() {
    sb, err := sandbox.New(sandbox.DefaultConfig())
    if err != nil {
        log.Fatal(err)
    }

    output, exitCode, err := sb.Run(context.Background(), "npm install")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Exit code: %d\nOutput:\n%s", exitCode, output)
}
```

### Custom Configuration

```go
sb, err := sandbox.New(sandbox.Config{
    Workdir:    "/path/to/project",
    AllowWrite: []string{"/path/to/project", "/tmp", "/var/cache"},
    DenyRead:   []string{"~/.ssh", "~/.aws", "~/.gnupg"},
    CleanEnv:   false,
})
```

### Filter Environment Variables

```go
// Block specific env vars from being passed to sandboxed process
sb, err := sandbox.New(sandbox.Config{
    Workdir:    "/path/to/project",
    AllowWrite: []string{"/path/to/project", "/tmp"},
    DenyRead:   []string{"~/.ssh", "~/.aws"},
    EnvDenylist: []string{
        "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
        "GITHUB_TOKEN", "GH_TOKEN",
        "OPENAI_API_KEY", "ANTHROPIC_API_KEY",
    },
})
```

### Clean Environment Mode

```go
// Start with minimal environment, only allow specific vars
sb, err := sandbox.New(sandbox.Config{
    Workdir:      "/path/to/project",
    AllowWrite:   []string{"/path/to/project", "/tmp"},
    CleanEnv:     true,
    EnvAllowlist: []string{"NODE_ENV", "CI"},  // Plus PATH, HOME, USER, TERM (always included)
})
```

### Pipe stdin

```go
input := strings.NewReader("hello world")
output, exitCode, err := sb.RunWithStdin(ctx, "cat", input)
```

### With Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, exitCode, err := sb.Run(ctx, "npm install")
if errors.Is(err, context.DeadlineExceeded) {
    log.Println("Command timed out")
}
```

### Dry Run

```go
sb, err := sandbox.New(sandbox.Config{
    Workdir: "/tmp",
    DryRun:  true,
})

output, _, _ := sb.Run(ctx, "rm -rf /")
fmt.Println(string(output))  // Prints the sandbox command that would be executed
```
