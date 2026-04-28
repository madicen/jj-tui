package ai

// Kind identifies which UI surface requested generation.
type Kind int

const (
	KindCommitDescription Kind = iota
	KindPR
	KindBookmark
	KindTicket
)

// TextGeneratedMsg is sent when an LLM request finishes (success or failure).
type TextGeneratedMsg struct {
	ReqID    int
	Kind     Kind
	CommitID string // change id for commit description correlation

	Text  string // commit description or bookmark name
	Title string
	Body  string
	Err   error
}
