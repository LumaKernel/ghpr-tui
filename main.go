package main

import (
	"fmt"
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

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version":
			fmt.Printf("ghpr-tui %s\n", version.Version)
			return
		case "-h", "--help":
			fmt.Print(helpText)
			return
		}
	}

	// Resolve repo from args or current directory
	repo := ""
	if len(os.Args) > 1 {
		repo = os.Args[1]
	}

	if repo == "" {
		var err error
		repo, err = ghclient.ResolveRepo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Run from a git repo or pass owner/repo as argument.\n")
			os.Exit(1)
		}
	}

	client := ghclient.NewClient(repo)

	store, err := state.NewStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing state: %v\n", err)
		os.Exit(1)
	}

	model := app.New(repo, client, store)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
