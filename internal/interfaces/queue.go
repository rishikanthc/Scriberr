package interfaces

// TaskQueueInterface defines the contract for task queue operations.
// This interface is already partially defined in dropzone/dropzone.go,
// but we centralize it here for reuse across packages.
type TaskQueueInterface interface {
	// EnqueueJob adds a job to the processing queue
	EnqueueJob(jobID string) error

	// KillJob terminates a running job
	KillJob(jobID string) error

	// IsJobRunning checks if a job is currently being processed
	IsJobRunning(jobID string) bool

	// GetQueueStats returns queue statistics
	GetQueueStats() map[string]interface{}
}
