package apiserver

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/service"
)

type DashboardProvider interface {
	GetDashboard(rangeKey string, timezone string) (dto.DashboardResponse, error)
}

type DashboardHandler struct {
	dashboard DashboardProvider
}

type DashboardQuery struct {
	Range string `form:"range"`
	TZ    string `form:"tz"`
}

func NewDashboardHandler(dashboard DashboardProvider) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

func (h *DashboardHandler) Get(c *gin.Context) {
	if h.dashboard == nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, "Dashboard service is not configured")
		return
	}

	var query DashboardQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}

	if query.Range == "" {
		query.Range = "30d"
	}

	if _, err := service.ParseDashboardRange(query.Range); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "range must be one of 7d, 30d, 90d")
		return
	}

	query.TZ, _ = service.ResolveDashboardLocation(query.TZ)

	response, err := h.dashboard.GetDashboard(query.Range, query.TZ)
	if err != nil {
		if errors.Is(err, service.ErrInvalidDashboardRange) {
			ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "range must be one of 7d, 30d, 90d")
			return
		}
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	ResponseSuccess(c, response)
}
