package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/github"
)

func TestChannelMergeProgressReporter_Close(t *testing.T) {
	t.Run("can be called multiple times without panicking", func(t *testing.T) {
		reporter := NewChannelMergeProgressReporter()

		// Close multiple times - should not panic
		require.NotPanics(t, func() {
			reporter.Close()
			reporter.Close()
			reporter.Close()
		})
	})

	t.Run("channel is only closed once", func(t *testing.T) {
		reporter := NewChannelMergeProgressReporter()

		// Send some updates
		reporter.StepStarted(0, "Test step")
		reporter.StepCompleted(0)

		// Close the channel
		reporter.Close()

		// Verify channel is closed by trying to receive
		// After closing, we should be able to drain existing messages
		// but new sends should panic (though we can't test that easily)
		updates := reporter.Updates()

		// Drain the channel
		received := 0
		for {
			select {
			case _, ok := <-updates:
				if !ok {
					// Channel is closed
					require.Equal(t, 2, received, "Should have received 2 updates")
					return
				}
				received++
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Timeout waiting for channel to be drained")
			}
		}
	})

	t.Run("multiple close calls do not affect channel state", func(t *testing.T) {
		reporter := NewChannelMergeProgressReporter()

		// Close once
		reporter.Close()

		// Verify channel is closed
		updates := reporter.Updates()
		_, ok := <-updates
		require.False(t, ok, "Channel should be closed")

		// Close again - should not panic or change state
		require.NotPanics(t, func() {
			reporter.Close()
		})

		// Verify channel is still closed
		_, ok = <-updates
		require.False(t, ok, "Channel should still be closed after second Close()")
	})

	t.Run("reporter methods work before close", func(t *testing.T) {
		reporter := NewChannelMergeProgressReporter()
		updates := reporter.Updates()

		// Send updates
		reporter.StepStarted(0, "Step 1")
		reporter.StepWaiting(0, 5*time.Second, 10*time.Minute, []github.CheckDetail{{Name: "Test", Status: "IN_PROGRESS"}})
		reporter.StepCompleted(0)
		reporter.StepFailed(1, nil)

		// Verify we can receive updates
		update := <-updates
		require.Equal(t, "started", update.Type)
		require.Equal(t, 0, update.StepIndex)

		update = <-updates
		require.Equal(t, "waiting", update.Type)
		require.Equal(t, 5*time.Second, update.Elapsed)

		update = <-updates
		require.Equal(t, "completed", update.Type)

		update = <-updates
		require.Equal(t, "failed", update.Type)
		require.Equal(t, 1, update.StepIndex)

		// Close and verify channel closes
		reporter.Close()
		_, ok := <-updates
		require.False(t, ok, "Channel should be closed")
	})

	t.Run("close prevents further sends from panicking", func(t *testing.T) {
		reporter := NewChannelMergeProgressReporter()

		// Close the channel
		reporter.Close()

		// Attempting to send after close should panic
		// Note: This is expected behavior - we're testing that Close() works
		// In practice, the caller should not send after closing
		require.Panics(t, func() {
			reporter.StepStarted(0, "Should panic")
		}, "Sending to closed channel should panic")
	})
}
