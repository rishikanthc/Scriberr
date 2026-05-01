package llm

import "context"

func sendStreamEvent(ctx context.Context, eventChan chan<- StreamEvent, event StreamEvent) error {
	select {
	case eventChan <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
