package agent

import (
	"sync"
	"testing"
)

func TestCollector_AckPollCount_SubtractsOnlySent(t *testing.T) {
	c := NewCollector()

	// 5 collects => pollCount == 5
	for i := 0; i < 5; i++ {
		_ = c.Collect()
	}

	// Simulate additional collects that happen "during send".
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			_ = c.Collect()
		}
	}()

	sent := c.PollCountSnapshot()
	wg.Wait()

	// pollCount is now sent + 3 (order may vary, but must be >= sent)
	c.AckPollCount(sent)

	remaining := c.PollCountSnapshot()
	if remaining < 0 {
		t.Fatalf("pollCount must not be negative, got %d", remaining)
	}
	// We expect that at least the concurrent collects weren't discarded.
	if remaining == 0 {
		t.Fatalf("expected remaining polls after ack; got %d", remaining)
	}
}

