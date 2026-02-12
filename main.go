package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui"
	"github.com/madicen/jj-tui/internal/version"
)

func main() {
	// Parse command-line flags
	demoMode := flag.Bool("demo", false, "Run in demo mode with mock services (for screenshots/testing)")
	flag.Parse()

	// Load saved configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: Could not load config: %v\n", err)
		cfg = &config.Config{}
	}

	// Apply saved config to environment (env vars take precedence)
	cfg.ApplyToEnvironment()

	// Initialize the TUI application
	ctx := context.Background()

	// Check for updates in background (non-blocking)
	version.CheckForUpdates(ctx)

	// Create the model (demo mode uses mock services)
	var model *tui.Model
	if *demoMode {
		model = tui.NewDemo(ctx)
	} else {
		model = tui.New(ctx)
	}
	defer model.Close()

	// Create the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
