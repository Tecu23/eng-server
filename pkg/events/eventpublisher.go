package events

import "sync"

// EventType represents the type of event
type EventType string

// Define event types
const (
	EventGameCreated      EventType = "GAME_CREATED"
	EventMoveProcessed    EventType = "MOVE_PROCESSED"
	EventEngineMoved      EventType = "ENGINE_MOVED"
	EventClockUpdated     EventType = "CLOCK_UPDATED"
	EventTimeUp           EventType = "TIME_UP"
	EventGameTerminated   EventType = "GAME_TERMINATED"
	EventConnectionClosed EventType = "CONNECTION_CLOSED"
)

// Event represents an event in the system
type Event struct {
	Type    EventType
	GameID  string // Optional, can be empty for non-game events
	Payload interface{}
}

// Handler is a function that processes events
type Handler func(event Event)

// Publisher is the central event publisher
type Publisher struct {
	mu          sync.RWMutex
	subscribers map[EventType][]Handler
}

// NewPublisher creates a new event publisher
func NewPublisher() *Publisher {
	return &Publisher{
		subscribers: make(map[EventType][]Handler),
	}
}

// Subscribe registers a handler for a specific event type
func (p *Publisher) Subscribe(eventType EventType, handler Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.subscribers[eventType] = append(p.subscribers[eventType], handler)
}

// Publish broadcasts an event to all subsribers
func (p *Publisher) Publish(event Event) {
	p.mu.RLock()
	handlers := p.subscribers[event.Type]
	p.mu.RUnlock()

	// Call all handlers
	for _, handler := range handlers {
		go handler(event) // Run handlers concurrently
	}
}

// SubscribeAll registers a handler for all event types
func (p *Publisher) SubscribeAll(handler Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Special event type for "all events"
	p.subscribers["*"] = append(p.subscribers["*"], handler)
}

// Publish broadcasts an event to all subscribers including "all events" handlers
func (p *Publisher) publish(event Event) {
	p.mu.RLock()
	handlers := p.subscribers[event.Type]
	allHandlers := p.subscribers["*"]
	p.mu.RUnlock()

	// Call specific event handlers
	for _, handler := range handlers {
		go handler(event)
	}

	// Call "all events" handlers
	for _, handler := range allHandlers {
		go handler(event)
	}
}
