# jj-tui Makefile

.PHONY: build test clean screenshots demo-repo help

# Default target
all: build

# Build the application
build:
	go build -o jj-tui .

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f jj-tui
	rm -rf fixtures/demo-repo

# Setup the demo repository for screenshots
demo-repo:
	@echo "Setting up demo repository..."
	./fixtures/setup-demo-repo.sh

# Generate all screenshots using VHS
# Requires: vhs (https://github.com/charmbracelet/vhs)
# VHS outputs frames to directories; we extract the last frame as the final screenshot
screenshots: build demo-repo
	@echo "Generating screenshots..."
	@mkdir -p screenshots
	@rm -rf screenshots/*.png
	vhs vhs/graph.tape
	vhs vhs/prs.tape
	vhs vhs/tickets.tape
	vhs vhs/branches.tape
	vhs vhs/settings.tape
	vhs vhs/help.tape
	@echo "Screenshots saved to screenshots/"

# Generate a demo GIF showing the TUI in action
demo-gif: build demo-repo
	@echo "Generating demo GIF..."
	@mkdir -p screenshots
	vhs vhs/all.tape
	@echo "Demo GIF saved to screenshots/demo.gif"

# Generate individual screenshots
screenshot-graph: build demo-repo
	vhs vhs/graph.tape

screenshot-prs: build demo-repo
	vhs vhs/prs.tape

screenshot-tickets: build demo-repo
	vhs vhs/tickets.tape

screenshot-branches: build demo-repo
	vhs vhs/branches.tape

screenshot-settings: build demo-repo
	vhs vhs/settings.tape

screenshot-help: build demo-repo
	vhs vhs/help.tape

# Run in demo mode (for manual testing)
demo: build demo-repo
	cd fixtures/demo-repo && ../../jj-tui --demo

# Install dependencies
deps:
	go mod tidy
	@echo "Note: VHS must be installed separately: brew install vhs"

# Help
help:
	@echo "jj-tui Makefile targets:"
	@echo "  build        - Build the application"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  demo-repo    - Setup demo repository for screenshots"
	@echo "  screenshots  - Generate all screenshots using VHS"
	@echo "  demo         - Run the app in demo mode"
	@echo "  deps         - Install/tidy dependencies"
	@echo "  help         - Show this help"

