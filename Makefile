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
	@cp "$$(ls -1 screenshots/graph.png/frame-*.png | tail -1)" screenshots/graph-final.png && rm -rf screenshots/graph.png && mv screenshots/graph-final.png screenshots/graph.png
	vhs vhs/prs.tape
	@cp "$$(ls -1 screenshots/prs.png/frame-*.png | tail -1)" screenshots/prs-final.png && rm -rf screenshots/prs.png && mv screenshots/prs-final.png screenshots/prs.png
	vhs vhs/tickets.tape
	@cp "$$(ls -1 screenshots/tickets.png/frame-*.png | tail -1)" screenshots/tickets-final.png && rm -rf screenshots/tickets.png && mv screenshots/tickets-final.png screenshots/tickets.png
	vhs vhs/branches.tape
	@cp "$$(ls -1 screenshots/branches.png/frame-*.png | tail -1)" screenshots/branches-final.png && rm -rf screenshots/branches.png && mv screenshots/branches-final.png screenshots/branches.png
	vhs vhs/settings.tape
	@cp "$$(ls -1 screenshots/settings.png/frame-*.png | tail -1)" screenshots/settings-final.png && rm -rf screenshots/settings.png && mv screenshots/settings-final.png screenshots/settings.png
	vhs vhs/help.tape
	@cp "$$(ls -1 screenshots/help.png/frame-*.png | tail -1)" screenshots/help-final.png && rm -rf screenshots/help.png && mv screenshots/help-final.png screenshots/help.png
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

