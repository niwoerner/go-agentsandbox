package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/niwoerner/go-agentsandbox/sandbox"
)

const exitSandboxError = 125 // Like docker

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "exec":
		execCmd(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func execCmd(args []string) {
	fs := flag.NewFlagSet("exec", flag.ExitOnError)

	var (
		configPath string
		noConfig   bool
		workdir    string
		allowWrite stringSlice
		denyRead   stringSlice
		cleanEnv   bool
		dryRun     bool
	)

	fs.StringVar(&configPath, "config", "", "Config file path (default: ~/.agent/sandbox/config.json)")
	fs.BoolVar(&noConfig, "no-config", false, "Skip loading config file")
	fs.StringVar(&workdir, "workdir", "", "Working directory (default: cwd)")
	fs.Var(&allowWrite, "allow-write", "Writable path, replaces config (repeatable)")
	fs.Var(&denyRead, "deny-read", "Protected path, replaces config (repeatable)")
	fs.BoolVar(&cleanEnv, "clean-env", false, "Start with minimal environment")
	fs.BoolVar(&dryRun, "dry-run", false, "Print command instead of executing")

	// Find -- separator
	cmdStart := -1
	for i, arg := range args {
		if arg == "--" {
			cmdStart = i
			break
		}
	}

	if cmdStart == -1 {
		fmt.Fprintln(os.Stderr, "error: missing -- before command")
		fmt.Fprintln(os.Stderr, "usage: agentsandbox exec [flags] -- COMMAND")
		os.Exit(exitSandboxError)
	}

	if err := fs.Parse(args[:cmdStart]); err != nil {
		os.Exit(exitSandboxError)
	}

	command := strings.Join(args[cmdStart+1:], " ")
	if command == "" {
		fmt.Fprintln(os.Stderr, "error: no command specified")
		os.Exit(exitSandboxError)
	}

	// Build config based on flags
	var cfg sandbox.Config
	if noConfig {
		// Skip config file, use hardcoded defaults only
		cfg = sandbox.DefaultConfigWithPath("")
	} else if configPath != "" {
		// Use specified config file
		cfg = sandbox.DefaultConfigWithPath(configPath)
	} else {
		// Use default config file path
		cfg = sandbox.DefaultConfig()
	}

	if workdir != "" {
		cfg.Workdir = workdir
	}

	// CLI flags replace config values (not append)
	if len(allowWrite) > 0 {
		cfg.AllowWrite = allowWrite
	}

	if len(denyRead) > 0 {
		cfg.DenyRead = denyRead
	}

	if cleanEnv {
		cfg.CleanEnv = true
	}
	cfg.DryRun = dryRun

	// Create sandbox
	sb, err := sandbox.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sandbox error: %v\n", err)
		os.Exit(exitSandboxError)
	}

	// Run command
	output, exitCode, err := sb.Run(context.Background(), command)

	// Print output
	os.Stdout.Write(output)

	if err != nil && exitCode == 0 {
		// Error but no exit code means sandbox issue
		fmt.Fprintf(os.Stderr, "execution error: %v\n", err)
		os.Exit(exitSandboxError)
	}

	os.Exit(exitCode)
}

func printUsage() {
	fmt.Println(`agentsandbox - filesystem sandbox for AI agents

Usage:
  agentsandbox exec [flags] -- COMMAND
  agentsandbox help

Commands:
  exec    Run a command in the sandbox
  help    Show this help

Flags for exec:
  --config PATH        Config file path (default: ~/.agent/sandbox/config.json)
  --no-config          Skip loading config file
  --workdir DIR        Working directory (default: cwd)
  --allow-write PATH   Writable path, replaces config (repeatable)
  --deny-read PATH     Protected path, replaces config (repeatable)
  --clean-env          Start with minimal environment
  --dry-run            Print command instead of executing

Config file format (JSON):
  {
    "allowWrite": ["/tmp", "."],
    "denyRead": ["~/.ssh", "~/.aws"],
    "cleanEnv": false,
    "envDenylist": ["AWS_SECRET_ACCESS_KEY"]
  }

Use "*" as wildcard: "allowWrite": ["*"] allows all writes.

Examples:
  agentsandbox exec -- npm install
  agentsandbox exec --workdir /project -- make build
  agentsandbox exec --config ./my-config.json -- make build
  agentsandbox exec --no-config -- ls -la
  agentsandbox exec --dry-run -- rm -rf /

Exit codes:
  0-124    Passed through from sandboxed command
  125      Sandbox setup or execution error`)
}
