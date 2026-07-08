package operations

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// operationLogLimit caps how many operations the browser loads. The op log can
// be very long in an active repo; the most recent operations are what matter
// for time-travel, and listnav scrolls within this window.
const operationLogLimit = 200

// OperationsLoadedMsg carries the result of loading `jj op log`.
type OperationsLoadedMsg struct {
	Operations []jj.Operation
	Err        error
}

// OperationRestoredMsg is sent after `jj op restore` completes. StatusMessage is
// a human-readable summary and Err is non-nil on failure. The main model reloads
// the repository graph on success.
type OperationRestoredMsg struct {
	StatusMessage string
	Err           error
}

// LoadOperationsCmd loads the operation log and sends OperationsLoadedMsg.
func LoadOperationsCmd(svc *jj.Service) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		ops, err := svc.ListOperations(context.Background(), operationLogLimit)
		return OperationsLoadedMsg{Operations: ops, Err: err}
	}
}

// RestoreOperationCmd runs `jj op restore <id>` and sends OperationRestoredMsg.
func RestoreOperationCmd(svc *jj.Service, opID string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		if err := svc.RestoreOperation(context.Background(), opID); err != nil {
			return OperationRestoredMsg{Err: err}
		}
		return OperationRestoredMsg{StatusMessage: "Restored to operation " + opID}
	}
}
