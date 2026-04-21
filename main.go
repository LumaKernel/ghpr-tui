package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghprq/internal/app"
	"github.com/LumaKernel/ghprq/internal/ghclient"
	"github.com/LumaKernel/ghprq/internal/state"
)

func main() {
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
