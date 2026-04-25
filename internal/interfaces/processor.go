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
