package stream

import (
	"github.com/vdibart/polis-cli/cli-go/pkg/discovery"
)

// ProjectionHandler defines how a projection processes stream events.
type ProjectionHandler interface {
	// TypePrefix returns the event type prefix this handler owns (e.g., "polis.follow").
	TypePrefix() string

	// EventTypes returns the specific event types this handler consumes.
	EventTypes() []string

	// NewState returns a zero-value state for this projection.
	NewState() interface{}

	// Process applies events to the current state and returns the updated state.
	Process(events []discovery.StreamEvent, state interface{}) (interface{}, error)
}

// BuiltinHandlers maps projection names to their built-in handler implementations.
// Note: NotificationHandler is not included here because it uses a different
// processing model (rule-driven, writes to state.jsonl instead of projection state).
var BuiltinHandlers = map[string]ProjectionHandler{
	"polis.follow":   &FollowHandler{},
	"polis.blessing": &BlessingHandler{},
}
