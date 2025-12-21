package tui

import (
	"sync"
	"time"
)

// ChannelMergeProgressReporter implements MergeProgressReporter using channels
type ChannelMergeProgressReporter struct {
	updates chan MergeProgressUpdate
	once    sync.Once
}

// NewChannelMergeProgressReporter creates a new channel-based progress reporter
func NewChannelMergeProgressReporter() *ChannelMergeProgressReporter {
	return &ChannelMergeProgressReporter{
		updates: make(chan MergeProgressUpdate, 100),
	}
}

// Updates returns the channel for receiving updates
func (r *ChannelMergeProgressReporter) Updates() <-chan MergeProgressUpdate {
	return r.updates
}

// Close closes the update channel (safe to call multiple times)
func (r *ChannelMergeProgressReporter) Close() {
	r.once.Do(func() {
		close(r.updates)
	})
}

// StepStarted reports that a step has started
func (r *ChannelMergeProgressReporter) StepStarted(stepIndex int, description string) {
	r.updates <- MergeProgressUpdate{
		Type:        "started",
		StepIndex:   stepIndex,
		Description: description,
	}
}

// StepCompleted reports that a step has completed
func (r *ChannelMergeProgressReporter) StepCompleted(stepIndex int) {
	r.updates <- MergeProgressUpdate{
		Type:      "completed",
		StepIndex: stepIndex,
	}
}

// StepFailed reports that a step has failed
func (r *ChannelMergeProgressReporter) StepFailed(stepIndex int, err error) {
	r.updates <- MergeProgressUpdate{
		Type:      "failed",
		StepIndex: stepIndex,
		Error:     err,
	}
}

// StepWaiting reports progress on a waiting step
func (r *ChannelMergeProgressReporter) StepWaiting(stepIndex int, elapsed, timeout time.Duration) {
	r.updates <- MergeProgressUpdate{
		Type:      "waiting",
		StepIndex: stepIndex,
		Elapsed:   elapsed,
		Timeout:   timeout,
	}
}
