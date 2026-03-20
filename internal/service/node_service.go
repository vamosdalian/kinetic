package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

type NodeService struct {
	db               database.Database
	runService       *RunService
	hub              *WorkerStreamHub
	heartbeatTimeout time.Duration
}

func NewNodeService(db database.Database, runService *RunService, hub *WorkerStreamHub, heartbeatTimeout time.Duration) *NodeService {
	return &NodeService{
		db:               db,
		runService:       runService,
		hub:              hub,
		heartbeatTimeout: heartbeatTimeout,
	}
}

func (s *NodeService) RegisterNode(req dto.RegisterNodeRequest) (dto.Node, error) {
	nodeID := strings.TrimSpace(req.NodeID)
	if nodeID == "" {
		return dto.Node{}, errors.New("node_id is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = nodeID
	}
	ip := strings.TrimSpace(req.IP)
	if req.MaxConcurrency <= 0 {
		req.MaxConcurrency = 1
	}
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = "remote"
	}

	now := time.Now()
	node := entity.NodeEntity{
		NodeID:          nodeID,
		Name:            name,
		IP:              ip,
		Kind:            kind,
		Status:          "online",
		MaxConcurrency:  req.MaxConcurrency,
		LastHeartbeatAt: &now,
		LastStreamAt:    &now,
	}
	if err := s.db.UpsertNode(node); err != nil {
		return dto.Node{}, err
	}

	if err := s.db.SaveNodeTag(entity.NodeTagEntity{NodeID: nodeID, Tag: "node-default", SystemManaged: true}); err != nil {
		return dto.Node{}, err
	}
	if ip != "" {
		if err := s.db.SaveNodeTag(entity.NodeTagEntity{NodeID: nodeID, Tag: fmt.Sprintf("node-%s", ip), SystemManaged: true}); err != nil {
			return dto.Node{}, err
		}
	}

	return s.GetNodeDTO(nodeID)
}

func (s *NodeService) Heartbeat(nodeID string) error {
	return s.db.UpdateNodeHeartbeat(nodeID)
}

func (s *NodeService) SubscribeStream(nodeID string) (<-chan dto.NodeCommand, func(), error) {
	if err := s.db.UpdateNodeStream(nodeID); err != nil {
		return nil, nil, err
	}
	ch, cleanup := s.hub.Subscribe(nodeID)
	return ch, func() {
		cleanup()
	}, nil
}

func (s *NodeService) ListNodeDTOs() ([]dto.Node, error) {
	nodes, err := s.db.ListNodes()
	if err != nil {
		return nil, err
	}

	result := make([]dto.Node, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, s.toNodeDTO(node))
	}
	return result, nil
}

func (s *NodeService) GetNodeDTO(nodeID string) (dto.Node, error) {
	node, err := s.db.GetNodeByID(nodeID)
	if err != nil {
		return dto.Node{}, err
	}
	return s.toNodeDTO(node), nil
}

func (s *NodeService) AddNodeTag(nodeID string, tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return errors.New("tag is required")
	}
	return s.db.SaveNodeTag(entity.NodeTagEntity{
		NodeID:        nodeID,
		Tag:           tag,
		SystemManaged: false,
	})
}

func (s *NodeService) DeleteNodeTag(nodeID string, tag string) error {
	return s.db.DeleteNodeTag(nodeID, tag)
}

func (s *NodeService) DispatchQueuedTasks(ctx context.Context, limit int) error {
	queued, err := s.db.ListQueuedTaskRuns(limit)
	if err != nil {
		return err
	}
	if len(queued) == 0 {
		return nil
	}

	nodes, err := s.db.ListNodes()
	if err != nil {
		return err
	}
	nodeMap := make(map[string]entity.NodeEntity, len(nodes))
	tagMap := make(map[string]map[string]bool, len(nodes))
	for _, node := range nodes {
		nodeMap[node.NodeID] = node
		tags, err := s.db.ListNodeTags(node.NodeID)
		if err != nil {
			return err
		}
		tagMap[node.NodeID] = make(map[string]bool, len(tags))
		for _, tag := range tags {
			tagMap[node.NodeID][tag.Tag] = true
		}
	}

	for _, task := range queued {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		selected, ok := s.selectNode(task.EffectiveTag, nodeMap, tagMap)
		if !ok {
			continue
		}

		if err := s.db.AssignTaskRun(task.RunID, task.TaskID, selected.NodeID); err != nil {
			return err
		}
		if err := s.db.IncrementNodeRunningCount(selected.NodeID); err != nil {
			_ = s.db.ResetAssignedTaskRun(task.RunID, task.TaskID)
			return err
		}
		selected.RunningCount++
		nodeMap[selected.NodeID] = selected

		assignedTask, err := s.runService.PrepareAssignedTask(task.RunID, task.TaskID)
		if err != nil {
			_ = s.db.DecrementNodeRunningCount(selected.NodeID)
			_ = s.db.ResetAssignedTaskRun(task.RunID, task.TaskID)
			selected.RunningCount--
			nodeMap[selected.NodeID] = selected
			return err
		}

		if ok := s.hub.Publish(selected.NodeID, dto.NodeCommand{Type: "assign", Task: assignedTask}); !ok {
			_ = s.db.DecrementNodeRunningCount(selected.NodeID)
			_ = s.db.ResetAssignedTaskRun(task.RunID, task.TaskID)
			_ = s.db.SetNodeStatus(selected.NodeID, "offline")
			selected.RunningCount--
			selected.Status = "offline"
			nodeMap[selected.NodeID] = selected
			continue
		}

		s.runService.publishTaskStatus(task.RunID, task.TaskID)
	}

	return nil
}

func (s *NodeService) SweepOfflineNodes(ctx context.Context) error {
	nodes, err := s.db.ListNodes()
	if err != nil {
		return err
	}

	now := time.Now()
	for _, node := range nodes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if node.Status != "online" {
			continue
		}
		// Heartbeats are the authoritative liveness signal. The worker stream is a
		// long-lived SSE connection whose timestamp is only refreshed when the
		// connection is established, so treating it as a continuously-updated
		// activity marker incorrectly marks healthy workers offline.
		if node.LastHeartbeatAt != nil && now.Sub(*node.LastHeartbeatAt) < s.heartbeatTimeout {
			continue
		}

		if err := s.db.SetNodeStatus(node.NodeID, "offline"); err != nil {
			return err
		}
		if err := s.runService.HandleNodeOffline(node.NodeID); err != nil {
			logrus.Warnf("failed to mark node %s tasks unknown: %v", node.NodeID, err)
		}
	}

	return nil
}

func (s *NodeService) HandleTaskEvent(nodeID string, event dto.WorkerTaskEvent) error {
	return s.runService.HandleWorkerTaskEvent(nodeID, event)
}

func (s *NodeService) PublishCancel(nodeID string, runID string, taskID string) {
	s.hub.Publish(nodeID, dto.NodeCommand{
		Type: "cancel",
		Task: &dto.AssignedTask{
			RunID:  runID,
			TaskID: taskID,
		},
	})
}

func (s *NodeService) toNodeDTO(node entity.NodeEntity) dto.Node {
	tags, _ := s.db.ListNodeTags(node.NodeID)
	dtoTags := make([]dto.NodeTag, 0, len(tags))
	for _, tag := range tags {
		dtoTags = append(dtoTags, dto.NodeTag{
			Tag:           tag.Tag,
			SystemManaged: tag.SystemManaged,
		})
	}
	sortNodeTags(dtoTags)
	return dto.Node{
		NodeID:          node.NodeID,
		Name:            node.Name,
		IP:              node.IP,
		Kind:            node.Kind,
		Status:          node.Status,
		MaxConcurrency:  node.MaxConcurrency,
		RunningCount:    node.RunningCount,
		LastHeartbeatAt: formatDTOTime(node.LastHeartbeatAt),
		LastStreamAt:    formatDTOTime(node.LastStreamAt),
		Tags:            dtoTags,
	}
}

func (s *NodeService) selectNode(requiredTag string, nodes map[string]entity.NodeEntity, tags map[string]map[string]bool) (entity.NodeEntity, bool) {
	var selected entity.NodeEntity
	found := false
	for _, node := range nodes {
		if node.Status != "online" {
			continue
		}
		if !s.hub.HasSubscriber(node.NodeID) {
			continue
		}
		if node.RunningCount >= node.MaxConcurrency {
			continue
		}
		if requiredTag != "" && !tags[node.NodeID][requiredTag] {
			continue
		}
		if !found || node.RunningCount < selected.RunningCount ||
			(node.RunningCount == selected.RunningCount && node.NodeID < selected.NodeID) {
			selected = node
			found = true
		}
	}

	return selected, found
}

func sortNodeTags(tags []dto.NodeTag) {
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Tag == tags[j].Tag {
			return !tags[i].SystemManaged && tags[j].SystemManaged
		}
		return tags[i].Tag < tags[j].Tag
	})
}

func formatDTOTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
