package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vamosdalian/kinetic/internal/model/dto"
)

func TestWorkerStreamHub_PublishReturnsFalseWhenSubscriberBufferIsFull(t *testing.T) {
	hub := NewWorkerStreamHub()
	stream, cleanup := hub.Subscribe("node-full")
	defer cleanup()

	hub.mu.RLock()
	listener := hub.listeners["node-full"][1]
	hub.mu.RUnlock()
	for cap(listener) > len(listener) {
		listener <- dto.NodeCommand{Type: "queued"}
	}

	assert.False(t, hub.Publish("node-full", dto.NodeCommand{Type: "assign"}))

	select {
	case command := <-stream:
		assert.Equal(t, "queued", command.Type)
	default:
		t.Fatal("expected buffered command")
	}
}

func TestWorkerStreamHub_PublishReturnsTrueWhenCommandIsDelivered(t *testing.T) {
	hub := NewWorkerStreamHub()
	stream, cleanup := hub.Subscribe("node-ready")
	defer cleanup()

	assert.True(t, hub.Publish("node-ready", dto.NodeCommand{Type: "assign"}))

	select {
	case command := <-stream:
		assert.Equal(t, "assign", command.Type)
	default:
		t.Fatal("expected published command")
	}
}
