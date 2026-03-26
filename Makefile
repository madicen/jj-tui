# jj-tui Makefile

.PHONY: build test clean screenshots demo-repo after-origin-vhs-repo after-origin-gif evolog-split-vhs-repo evolog-split-gif divergent-vhs-repo divergent-gif bookmark-conflict-vhs-repo bookmark-conflict-gif screenshot-after-origin screenshot-evolog-split screenshot-divergent screenshot-bookmark-conflict help

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
	rm -rf fixtures/demo-repo fixtures/after-origin-vhs-repo fixtures/after-origin-fake-origin.git fixtures/evolog-split-vhs-repo fixtures/divergent-vhs-repo fixtures/bookmark-conflict-vhs-repo fixtures/bookmark-conflict-fake-origin.git

# Setup the demo repository for screenshots
demo-repo:
	@echo "Setting up demo repository..."
	./fixtures/setup-demo-repo.sh

# Generate all screenshots using VHS
# Requires: vhs (https://github.com/charmbracelet/vhs)
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
	vhs vhs/command_history.tape
	vhs vhs/after-origin.tape
	vhs vhs/evolog-split.tape
	vhs vhs/divergent.tape
	vhs vhs/bookmark-conflict.tape
	@echo "Screenshots saved to screenshots/ (including after-origin.gif, evolog-split.gif, divergent.gif, bookmark-conflict.gif)"

# Generate a demo GIF showing the TUI in action
demo-gif: build demo-repo
	@echo "Generating demo GIF..."
	@mkdir -p screenshots
	vhs vhs/all.tape
	@echo "Demo GIF saved to screenshots/demo.gif"

# Generate demo GIF while recording CPU and memory profiles (profiles written on exit)
demo-gif-profile: build demo-repo
	@echo "Generating demo GIF with profiling..."
	@mkdir -p screenshots
	@rm -f screenshots/cpu.prof screenshots/mem.prof
	vhs vhs/all-profile.tape
	@echo "Demo GIF saved to screenshots/demo.gif"
	@if [ -f screenshots/cpu.prof ]; then echo "CPU profile: screenshots/cpu.prof (go tool pprof screenshots/cpu.prof)"; fi

# Dedicated GIF for graph: Forgot New Commit? (f) + Update PR (u) (see vhs/after-origin.tape)
after-origin-vhs-repo:
	bash fixtures/setup-after-origin-vhs-repo.sh

after-origin-gif: build after-origin-vhs-repo
	@mkdir -p screenshots
	vhs vhs/after-origin.tape
	@echo "GIF saved to screenshots/after-origin.gif"
	@if [ -f screenshots/mem.prof ]; then echo "Memory profile: screenshots/mem.prof (go tool pprof screenshots/mem.prof)"; fi

# Dedicated GIF for experimental Evolog split (z); see vhs/evolog-split.tape
evolog-split-vhs-repo:
	bash fixtures/setup-evolog-split-vhs-repo.sh

evolog-split-gif: build evolog-split-vhs-repo
	@mkdir -p screenshots
	vhs vhs/evolog-split.tape
	@echo "GIF saved to screenshots/evolog-split.gif"

# Resolving a divergent change (d); see vhs/divergent.tape
divergent-vhs-repo:
	bash fixtures/setup-divergent-vhs-repo.sh

divergent-gif: build divergent-vhs-repo
	@mkdir -p screenshots
	vhs vhs/divergent.tape
	@echo "GIF saved to screenshots/divergent.gif"

# Diverged bookmark: local tip != origin after amend; resolve via Branches (c) or graph (Shift+C); see vhs/bookmark-conflict.tape
bookmark-conflict-vhs-repo:
	bash fixtures/setup-bookmark-conflict-vhs-repo.sh

bookmark-conflict-gif: build bookmark-conflict-vhs-repo
	@mkdir -p screenshots
	vhs vhs/bookmark-conflict.tape
	@echo "GIF saved to screenshots/bookmark-conflict.gif"

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

screenshot-command-history: build demo-repo
	vhs vhs/command_history.tape

screenshot-after-origin: build after-origin-vhs-repo
	@mkdir -p screenshots
	vhs vhs/after-origin.tape

screenshot-evolog-split: build evolog-split-vhs-repo
	@mkdir -p screenshots
	vhs vhs/evolog-split.tape

screenshot-divergent: build divergent-vhs-repo
	@mkdir -p screenshots
	vhs vhs/divergent.tape

screenshot-bookmark-conflict: build bookmark-conflict-vhs-repo
	@mkdir -p screenshots
	vhs vhs/bookmark-conflict.tape

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
	@echo "  screenshots  - Generate PNG screenshots + after-origin.gif + evolog-split.gif (see also demo-gif)"
	@echo "  demo-gif     - Generate demo GIF (vhs all.tape)"
	@echo "  demo-gif-profile - Generate demo GIF with CPU/memory profiling"
	@echo "  after-origin-gif - Generate after-origin GIF: (f) restack + (u) update PR (vhs/after-origin.tape)"
	@echo "  evolog-split-gif - Generate evolog-split GIF: experimental (z) split (vhs/evolog-split.tape)"
	@echo "  divergent-gif    - Generate divergent GIF: resolve duplicate change ID (vhs/divergent.tape)"
	@echo "  bookmark-conflict-gif - Generate diverged-bookmark GIF: Branches (c) resolver (vhs/bookmark-conflict.tape)"
	@echo "  screenshot-after-origin   - Regenerate only screenshots/after-origin.gif"
	@echo "  screenshot-evolog-split   - Regenerate only screenshots/evolog-split.gif"
	@echo "  screenshot-divergent      - Regenerate only screenshots/divergent.gif"
	@echo "  screenshot-bookmark-conflict - Regenerate only screenshots/bookmark-conflict.gif"
	@echo "  demo         - Run the app in demo mode"
	@echo "  deps         - Install/tidy dependencies"
	@echo "  help         - Show this help"

