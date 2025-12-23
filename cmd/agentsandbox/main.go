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
		workdir    string
		allowWrite stringSlice
		denyRead   stringSlice
		cleanEnv   bool
		dryRun     bool
	)

	fs.StringVar(&workdir, "workdir", "", "Working directory (default: cwd)")
	fs.Var(&allowWrite, "allow-write", "Additional writable path (repeatable)")
	fs.Var(&denyRead, "deny-read", "Additional protected path (repeatable)")
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

	// Build config
	cfg := sandbox.DefaultConfig()

	if workdir != "" {
		cfg.Workdir = workdir
	}

	if len(allowWrite) > 0 {
		cfg.AllowWrite = append(cfg.AllowWrite, allowWrite...)
	}

	if len(denyRead) > 0 {
		cfg.DenyRead = append(cfg.DenyRead, denyRead...)
	}

	cfg.CleanEnv = cleanEnv
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
  --workdir DIR        Working directory (default: cwd)
  --allow-write PATH   Additional writable path (repeatable)
  --deny-read PATH     Additional protected path (repeatable)
  --clean-env          Start with minimal environment
  --dry-run            Print command instead of executing

Examples:
  agentsandbox exec -- npm install
  agentsandbox exec --workdir /project -- make build
  agentsandbox exec --allow-write /var/cache -- apt-get update
  agentsandbox exec --dry-run -- rm -rf /

Exit codes:
  0-124    Passed through from sandboxed command
  125      Sandbox setup or execution error`)
}
