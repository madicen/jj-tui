package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui"
	"github.com/madicen/jj-tui/internal/version"
)

func main() {
	// Parse command-line flags
	demoMode := flag.Bool("demo", false, "Run in demo mode with mock services (for screenshots/testing)")
	cpuProfile := flag.String("cpuprofile", "", "Write CPU profile to file (on exit)")
	memProfile := flag.String("memprofile", "", "Write memory profile to file (on exit)")
	pprofAddr := flag.String("pprof", "", "Serve pprof HTTP at address (e.g. :6060); use with -demo to profile live")
	flag.Parse()

	// Start CPU profiling if requested
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cpuprofile: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "cpuprofile: %v\n", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	// Serve pprof HTTP for live profiling (e.g. go tool pprof http://localhost:6060/debug/pprof/heap)
	if *pprofAddr != "" {
		go func() {
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				fmt.Fprintf(os.Stderr, "pprof server: %v\n", err)
			}
		}()
	}

	// Write memory profile on exit if requested
	if *memProfile != "" {
		defer func() {
			f, err := os.Create(*memProfile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "memprofile: %v\n", err)
				return
			}
			defer f.Close()
			if err := pprof.WriteHeapProfile(f); err != nil {
				fmt.Fprintf(os.Stderr, "memprofile: %v\n", err)
			}
		}()
	}

	// Force color output in demo mode (for VHS screenshots)
	// This ensures colors render even when TTY detection fails
	if *demoMode || os.Getenv("FORCE_COLOR") == "1" {
		lipgloss.SetColorProfile(termenv.TrueColor)
	}

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

	// Create the Bubble Tea program.
	// Use WithMouseAllMotion so wheel and motion work without requiring a prior click
	// (some terminals only deliver mouse events reliably in this mode).
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),        // Use alternate screen buffer
		tea.WithMouseAllMotion(),   // Mouse click, release, wheel, and motion without button press
	)

	// Run the program
	_, err = p.Run()

	// Disable xterm mouse tracking so the shell doesn't echo a mouse report (e.g. "35;270;24M") after exit.
	termenv.DefaultOutput().DisableMouseAllMotion()

	if err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
