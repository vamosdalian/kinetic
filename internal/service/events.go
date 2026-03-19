package service

import (
	"context"

	"github.com/vamosdalian/kinetic/internal/model/dto"
)

func (s *RunService) SubscribeRunEvents(runID string) (<-chan dto.WorkflowRunEvent, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSubscriberID++
	subscriberID := s.nextSubscriberID

	if s.subscribers[runID] == nil {
		s.subscribers[runID] = make(map[int]chan dto.WorkflowRunEvent)
	}

	ch := make(chan dto.WorkflowRunEvent, 256)
	s.subscribers[runID][subscriberID] = ch

	cleanup := func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		runSubscribers := s.subscribers[runID]
		if runSubscribers == nil {
			return
		}
		if subscriber, ok := runSubscribers[subscriberID]; ok {
			delete(runSubscribers, subscriberID)
			close(subscriber)
		}
		if len(runSubscribers) == 0 {
			delete(s.subscribers, runID)
		}
	}

	return ch, cleanup, nil
}

func (s *RunService) publishEvent(event dto.WorkflowRunEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, subscriber := range s.subscribers[event.RunID] {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (s *RunService) storeCancel(runID string, cancel context.CancelCauseFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancels[runID] = cancel
}

func (s *RunService) takeCancel(runID string) context.CancelCauseFunc {
	s.mu.Lock()
	defer s.mu.Unlock()
	cancel := s.cancels[runID]
	delete(s.cancels, runID)
	return cancel
}

func (s *RunService) getCancel(runID string) context.CancelCauseFunc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cancels[runID]
}
