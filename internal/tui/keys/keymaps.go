package keys

// KeyMaps aggregates every scope's KeyMap. The help tab reads it to generate the
// shortcuts list from the same bindings the handlers match against, and P5.1
// constructs it once from config so a rebinding is reflected everywhere.
type KeyMaps struct {
	Global   GlobalKeyMap
	Graph    GraphKeyMap
	Branches BranchesKeyMap
	PRs      PRsKeyMap
	Tickets  TicketsKeyMap
	Help     HelpKeyMap
}

// DefaultKeyMaps builds every scope's KeyMap, applying config overrides (nil for
// pure defaults).
func DefaultKeyMaps(overrides map[string]string) KeyMaps {
	return KeyMaps{
		Global:   DefaultGlobalKeyMap(overrides),
		Graph:    DefaultGraphKeyMap(overrides),
		Branches: DefaultBranchesKeyMap(overrides),
		PRs:      DefaultPRsKeyMap(overrides),
		Tickets:  DefaultTicketsKeyMap(overrides),
		Help:     DefaultHelpKeyMap(overrides),
	}
}
