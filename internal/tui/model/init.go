package model

import (
	"context"

	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
	bookmarktab "github.com/madicen/jj-tui/internal/tui/tabs/bookmark"
	branchestab "github.com/madicen/jj-tui/internal/tui/tabs/branches"
	conflicttab "github.com/madicen/jj-tui/internal/tui/tabs/conflict"
	descedittab "github.com/madicen/jj-tui/internal/tui/tabs/descedit"
	divergenttab "github.com/madicen/jj-tui/internal/tui/tabs/divergent"
	errortab "github.com/madicen/jj-tui/internal/tui/tabs/error"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
	githublogintab "github.com/madicen/jj-tui/internal/tui/tabs/githublogin"
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
	initrepotab 	"github.com/madicen/jj-tui/internal/tui/tabs/initrepo"
	prformtab "github.com/madicen/jj-tui/internal/tui/tabs/prform"
	prstab "github.com/madicen/jj-tui/internal/tui/tabs/prs"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
	ticketstab "github.com/madicen/jj-tui/internal/tui/tabs/tickets"
	warningtab "github.com/madicen/jj-tui/internal/tui/tabs/warning"
)

// New creates a new Model
func New(ctx context.Context) *Model {
	// Load config for initial values
	cfg, _ := config.Load()

	zm := zone.New()
	graphTabModel := graphtab.NewGraphModel(zm)

	settingsTabModel := settingstab.NewModelWithConfig(cfg)

	m := &Model{
		ctx:              ctx,
		zoneManager:      zm,
		appState: state.AppState{
			ViewMode:      state.ViewCommitGraph,
			StatusMessage: "Initializing...",
			Loading:       true,
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
		bookmarkModal:    bookmarktab.NewModel(zm),
		prFormModal:      prformtab.NewModel(zm),
		desceditModal:    descedittab.NewModel(zm),
		githubLoginModel: githublogintab.NewModel(zm),
	}
	m.errorModal.SetZoneManager(zm)
	m.initRepoModel.SetZoneManager(zm)
	m.warningModal.SetZoneManager(zm)
	m.settingsTabModel.SetZoneManager(zm)
	m.githubLoginModel.SetZoneManager(zm)
	m.appState.Config = cfg
	return m
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
