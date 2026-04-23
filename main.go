package main

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghpr-tui/internal/app"
	"github.com/LumaKernel/ghpr-tui/internal/ghclient"
	"github.com/LumaKernel/ghpr-tui/internal/state"
	"github.com/LumaKernel/ghpr-tui/internal/version"
)

const helpText = `ghpr-tui - TUI for reviewing GitHub Pull Requests

Usage:
  ghpr-tui [owner/repo]    Open TUI for the given repo (or current repo)
  ghpr-tui -v, --version   Show version
  ghpr-tui -h, --help      Show this help

Key bindings (PR list):
  j/k         Navigate          enter    Open PR
  c           Checks            m        Merge
  d           Toggle draft      r        Toggle read
  R           Refresh           o        Open in browser
  C-d/C-u     Half-page scroll  q        Quit

Key bindings (file list):
  j/k         Navigate          enter    View diff
  space       Toggle reviewed   a        Mark all reviewed
  c           Checks            m        Merge
  esc         Back

Key bindings (diff view):
  j/k         Scroll            d/u      Half-page
  f/b         Full page         n/N      Next/prev hunk
  ]/[         Next/prev file    space    Reviewed + next
  esc         Back
`

// parseAction represents the result of parsing CLI arguments.
type parseAction int

const (
	actionRun     parseAction = iota // run the TUI
	actionVersion                    // print version
	actionHelp                       // print help
)

// parseArgs parses CLI arguments and returns the action and repo string.
func parseArgs(args []string) (parseAction, string) {
	if len(args) > 1 {
		switch args[1] {
		case "-v", "--version":
			return actionVersion, ""
		case "-h", "--help":
			return actionHelp, ""
		default:
			return actionRun, args[1]
		}
	}
	return actionRun, ""
}

// printVersion writes the version string to the given writer.
func printVersion(w io.Writer) {
	fmt.Fprintf(w, "ghpr-tui %s\n", version.Version)
}

// printHelp writes the help text to the given writer.
func printHelp(w io.Writer) {
	fmt.Fprint(w, helpText)
}

// setup handles CLI args, resolves repo, and creates the app model.
// Returns nil model if version/help was printed (no TUI needed).
func setup(args []string, stdout, stderr io.Writer) (tea.Model, error) {
	action, repo := parseArgs(args)

	switch action {
	case actionVersion:
		printVersion(stdout)
		return nil, nil
	case actionHelp:
		printHelp(stdout)
		return nil, nil
	case actionRun:
		// continue
	}

	client := ghclient.NewClient(repo)

	if repo == "" {
		var err error
		repo, err = client.ResolveRepo()
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			fmt.Fprintf(stderr, "Run from a git repo or pass owner/repo as argument.\n")
			return nil, err
		}
	}

	store, err := state.NewStore()
	if err != nil {
		fmt.Fprintf(stderr, "Error initializing state: %v\n", err)
		return nil, err
	}

	return app.New(repo, client, store), nil
}

// run executes the main application logic and returns an error instead of calling os.Exit.
func run(args []string, stdout, stderr io.Writer) error {
	model, err := setup(args, stdout, stderr)
	if err != nil {
		return err
	}
	if model == nil {
		return nil
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return err
	}
	return nil
}

func main() {
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
