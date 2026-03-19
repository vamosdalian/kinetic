package service

import (
	"sync"

	"github.com/vamosdalian/kinetic/internal/model/dto"
)

type WorkerStreamHub struct {
	mu        sync.RWMutex
	nextID    int
	listeners map[string]map[int]chan dto.NodeCommand
}

func NewWorkerStreamHub() *WorkerStreamHub {
	return &WorkerStreamHub{
		listeners: make(map[string]map[int]chan dto.NodeCommand),
	}
}

func (h *WorkerStreamHub) Subscribe(nodeID string) (<-chan dto.NodeCommand, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	subID := h.nextID
	if h.listeners[nodeID] == nil {
		h.listeners[nodeID] = make(map[int]chan dto.NodeCommand)
	}

	ch := make(chan dto.NodeCommand, 64)
	h.listeners[nodeID][subID] = ch

	cleanup := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		listeners := h.listeners[nodeID]
		if listeners == nil {
			return
		}
		if sub, ok := listeners[subID]; ok {
			delete(listeners, subID)
			close(sub)
		}
		if len(listeners) == 0 {
			delete(h.listeners, nodeID)
		}
	}

	return ch, cleanup
}

func (h *WorkerStreamHub) Publish(nodeID string, command dto.NodeCommand) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	listeners := h.listeners[nodeID]
	if len(listeners) == 0 {
		return false
	}

	for _, listener := range listeners {
		select {
		case listener <- command:
		default:
		}
	}
	return true
}

func (h *WorkerStreamHub) HasSubscriber(nodeID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.listeners[nodeID]) > 0
}
