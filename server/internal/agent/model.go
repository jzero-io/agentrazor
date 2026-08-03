package agent

const (
	RunStatusQueued    = "queued"
	RunStatusRunning   = "running"
	RunStatusCompleted = "completed"
	RunStatusFailed    = "failed"
	RunStatusCancelled = "cancelled"
	RunStatusAborted   = "aborted"
)

type RuntimeResult struct {
	ExternalSessionID string
	Output            string
	ExitCode          int
	Events            []map[string]any
}
