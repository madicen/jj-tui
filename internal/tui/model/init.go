package model

import (
	"context"

	zone "github.com/lrstanley/bubblezone"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/tab"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	conflicttab "github.com/madicen/jj-tui/internal/tui/tabs/conflict"
	descedittab "github.com/madicen/jj-tui/internal/tui/tabs/descedit"
	divergenttab "github.com/madicen/jj-tui/internal/tui/tabs/divergent"
	errortab "github.com/madicen/jj-tui/internal/tui/tabs/error"
	evologsplittab "github.com/madicen/jj-tui/internal/tui/tabs/evologsplit"
	filedifftab "github.com/madicen/jj-tui/internal/tui/tabs/filediff"
	githublogintab "github.com/madicen/jj-tui/internal/tui/tabs/githublogin"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
	initrepotab "github.com/madicen/jj-tui/internal/tui/tabs/initrepo"
	prformtab "github.com/madicen/jj-tui/internal/tui/tabs/prform"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
	ticketformtab "github.com/madicen/jj-tui/internal/tui/tabs/ticketform"
	ticketstab "github.com/madicen/jj-tui/internal/tui/tabs/tickets"
	warningtab "github.com/madicen/jj-tui/internal/tui/tabs/warning"
	workspacestab "github.com/madicen/jj-tui/internal/tui/tabs/workspaces"
)

// New creates a new Model
func New(ctx context.Context) *Model {
	// Load config for initial values
	cfg, _ := config.Load()

	zm := zone.New()
	graphTabModel := graphtab.NewGraphModel(zm)

	settingsTabModel := settingstab.NewModelWithConfig(cfg)

	m := &Model{
		ctx:         ctx,
		zoneManager: zm,
		busySpinner: newBusySpinner(),
		appState: state.AppState{
			ViewMode:      state.ViewCommitGraph,
			StatusMessage: "Initializing...",
			Loading:       false,
		},
		graphTabModel:    graphTabModel,
		prsTabModel:      prstab.NewModel(zm),
		branchesTabModel: branchestab.NewModel(zm),
		ticketsTabModel:  ticketstab.NewModel(zm),
		settingsTabModel: settingsTabModel,
		helpTabModel:     helptab.NewModel(zm),
		initRepoModel:    initrepotab.NewModel(),
		errorModal:       errortab.NewModel(),
		warningModal:     warningtab.NewModel(),
		conflictModal:    conflicttab.NewModel(zm),
		divergentModal:   divergenttab.NewModel(zm),
		evologSplitModal: evologsplittab.NewModel(zm),
		fileDiffModal:    filedifftab.NewModel(zm),
		bookmarkModal:    bookmarktab.NewModel(zm),
		prFormModal:      prformtab.NewModel(zm),
		ticketFormModal:  ticketformtab.NewModel(zm),
		desceditModal:    descedittab.NewModel(zm),
		githubLoginModel: githublogintab.NewModel(zm),
		workspacesModal:  workspacestab.NewModel(),
	}
	m.errorModal.SetZoneManager(zm)
	m.initRepoModel.SetZoneManager(zm)
	m.warningModal.SetZoneManager(zm)
	m.settingsTabModel.SetZoneManager(zm)
	m.githubLoginModel.SetZoneManager(zm)
	m.appState.Config = cfg
	// ShowMinimizeButton renders a [-]/[+] toggle in the chrome tab so the
	// user can collapse any chromed modal to its title strip and click on the
	// underlying tab (graph / PRs / branches / tickets) while the modal stays
	// open in the background. mouseTransparent() pairs with this by routing
	// off-modal clicks to the underlay view once LayerState.Minimized flips.
	m.chrome.Configure = func(c *overlay.OverlayConfig) {
		c.WindowChrome.ShowMinimizeButton = true
	}
	m.initTabRegistry()
	return m
}

// initTabRegistry wires the six primary content tabs into the tab.Renderer
// registry. Entries point at the concrete struct fields so reassignments to
// those fields (e.g. m.graphTabModel = updated) stay visible through the
// registry. tabOrder preserves the fan-out order the resize handler used before
// the registry existed.
func (m *Model) initTabRegistry() {
	m.tabOrder = []state.ViewMode{
		state.ViewCommitGraph,
		state.ViewPullRequests,
		state.ViewBranches,
		state.ViewTickets,
		state.ViewSettings,
		state.ViewHelp,
	}
	m.tabRegistry = map[state.ViewMode]tab.Renderer{
		state.ViewCommitGraph:  &m.graphTabModel,
		state.ViewPullRequests: &m.prsTabModel,
		state.ViewBranches:     &m.branchesTabModel,
		state.ViewTickets:      &m.ticketsTabModel,
		state.ViewSettings:     &m.settingsTabModel,
		state.ViewHelp:         &m.helpTabModel,
	}
}

// NewWithServices creates a new Model with pre-configured services
func NewWithServices(ctx context.Context, jjSvc *jj.Service, ghSvc *github.Service) *Model {
	m := New(ctx)
	m.appState.JJService = jjSvc
	m.appState.GitHubService = ghSvc
	return m
}

// NewDemo creates a new Model in demo mode with mock services
// This is used for VHS screenshots and visual testing
func NewDemo(ctx context.Context) *Model {
	m := New(ctx)
	m.appState.DemoMode = true
	return m
}
