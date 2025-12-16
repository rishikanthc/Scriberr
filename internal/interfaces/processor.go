package interfaces

import (
	"context"
	"os/exec"
)

// JobProcessorInterface defines the contract for job processing.
// This matches the existing queue.JobProcessor interface but is centralized here.
type JobProcessorInterface interface {
	// ProcessJob processes a transcription job
	ProcessJob(ctx context.Context, jobID string) error

	// ProcessJobWithProcess processes a job and allows registering the process for termination
	ProcessJobWithProcess(ctx context.Context, jobID string, registerProcess func(*exec.Cmd)) error
}

// MultiTrackJobProcessorInterface extends JobProcessorInterface with multi-track support.
type MultiTrackJobProcessorInterface interface {
	JobProcessorInterface

	// TerminateMultiTrackJob terminates a multi-track job and all its individual track jobs
	TerminateMultiTrackJob(jobID string) error

	// IsMultiTrackJob checks if a job is a multi-track job
	IsMultiTrackJob(jobID string) bool
}
