package lirc

import (
	"context"
	"path/filepath"
)

type RemoteHandlers map[string]ButtonHandlers
type ButtonHandlers map[string]ButtonHandler
type ButtonHandler func(ButtonPress)

// RouteEvents routes events to the appropriate handler until ctx is canceled.
func RouteEvents(ctx context.Context, events <-chan ButtonPress, handlers RemoteHandlers) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event := <-events:
			// Check for exact match
			if h := handlers[event.RemoteControlName][event.ButtonName]; h != nil {
				h(event)
				continue
			}

			// Check for pattern matches
			for remote, buttonHandlers := range handlers {
				remoteMatched, _ := filepath.Match(remote, event.RemoteControlName)
				if !remoteMatched {
					continue
				}

				for button, h := range buttonHandlers {
					buttonMatched, _ := filepath.Match(button, event.ButtonName)
					if !buttonMatched {
						continue
					}
					h(event)
				}
			}
		}
	}
}
