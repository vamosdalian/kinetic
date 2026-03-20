package apiserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vamosdalian/kinetic/internal/model/dto"
)

type NodeManager interface {
	RegisterNode(req dto.RegisterNodeRequest) (dto.Node, error)
	Heartbeat(nodeID string) error
	SubscribeStream(nodeID string) (<-chan dto.NodeCommand, func(), error)
	HandleTaskEvent(nodeID string, event dto.WorkerTaskEvent) error
	ListNodeDTOs() ([]dto.Node, error)
	GetNodeDTO(nodeID string) (dto.Node, error)
	AddNodeTag(nodeID string, tag string) error
	DeleteNodeTag(nodeID string, tag string) error
}

type NodeHandler struct {
	nodes NodeManager
}

type NodeListQuery struct {
	PageQuerys
	Query string `form:"query"`
}

func NewNodeHandler(nodes NodeManager) *NodeHandler {
	return &NodeHandler{nodes: nodes}
}

func (h *NodeHandler) List(c *gin.Context) {
	var query NodeListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	nodes, err := h.nodes.ListNodeDTOs()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	filtered := filterNodeDTOs(nodes, query.Query)
	start := (query.Page - 1) * query.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + query.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	ResponseSuccessWithPagination(c, filtered[start:end], query.Page, query.PageSize, len(filtered))
}

func filterNodeDTOs(nodes []dto.Node, query string) []dto.Node {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return nodes
	}

	filtered := make([]dto.Node, 0, len(nodes))
	for _, node := range nodes {
		parts := []string{node.NodeID, node.Name, node.IP, string(node.Kind), string(node.Status)}
		for _, tag := range node.Tags {
			parts = append(parts, tag.Tag)
		}
		if strings.Contains(strings.ToLower(strings.Join(parts, " ")), needle) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

func (h *NodeHandler) Get(c *gin.Context) {
	nodeID := c.Param("id")
	node, err := h.nodes.GetNodeDTO(nodeID)
	if err != nil {
		ResponseError(c, http.StatusNotFound, ErrorCodeInvalidRequest, err.Error())
		return
	}
	ResponseSuccess(c, node)
}

func (h *NodeHandler) AddTag(c *gin.Context) {
	nodeID := c.Param("id")
	var req dto.AddNodeTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	if err := h.nodes.AddNodeTag(nodeID, req.Tag); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	ResponseSuccess(c, gin.H{"node_id": nodeID, "tag": req.Tag})
}

func (h *NodeHandler) DeleteTag(c *gin.Context) {
	nodeID := c.Param("id")
	tag := c.Param("tag")
	if err := h.nodes.DeleteNodeTag(nodeID, tag); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	ResponseSuccess(c, gin.H{"node_id": nodeID, "tag": tag})
}

func (h *NodeHandler) Register(c *gin.Context) {
	var req dto.RegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	if req.IP == "" {
		req.IP = c.ClientIP()
	}
	node, err := h.nodes.RegisterNode(req)
	if err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	ResponseSuccess(c, node)
}

func (h *NodeHandler) Heartbeat(c *gin.Context) {
	nodeID := c.Param("id")
	if err := h.nodes.Heartbeat(nodeID); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	ResponseSuccess(c, gin.H{"node_id": nodeID})
}

func (h *NodeHandler) Stream(c *gin.Context) {
	nodeID := c.Param("id")
	stream, cleanup, err := h.nodes.SubscribeStream(nodeID)
	if err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	defer cleanup()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	disableStreamingWriteDeadline(c)

	keepalive := time.NewTicker(10 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case command, ok := <-stream:
			if !ok {
				return
			}
			writeSSEEvent(c, command.Type, command)
		case <-keepalive.C:
			writeSSEEvent(c, "keepalive", gin.H{"ts": time.Now().Format(time.RFC3339)})
		}
	}
}

func (h *NodeHandler) TaskEvents(c *gin.Context) {
	nodeID := c.Param("id")
	var event dto.WorkerTaskEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	if err := h.nodes.HandleTaskEvent(nodeID, event); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	ResponseSuccess(c, gin.H{"node_id": nodeID})
}
