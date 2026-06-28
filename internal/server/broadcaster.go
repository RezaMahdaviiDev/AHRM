package server

import "sync"

// Broadcaster fan-outs string events to all subscribed SSE clients.
type Broadcaster struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{clients: make(map[chan string]struct{})}
}

func (b *Broadcaster) Subscribe() chan string {
	ch := make(chan string, 4)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Broadcaster) Unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

// Publish sends msg to all connected clients; slow clients are dropped (non-blocking).
func (b *Broadcaster) Publish(msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}
