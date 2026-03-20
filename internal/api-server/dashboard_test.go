package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vamosdalian/kinetic/internal/model/dto"
)

type stubDashboardProvider struct {
	rangeKey string
	timezone string
}

func (s *stubDashboardProvider) GetDashboard(rangeKey string, timezone string) (dto.DashboardResponse, error) {
	s.rangeKey = rangeKey
	s.timezone = timezone
	return dto.DashboardResponse{
		Chart: dto.DashboardChart{
			Range:    rangeKey,
			Timezone: timezone,
			Points:   []dto.DashboardChartPoint{},
		},
		Tables: dto.DashboardTables{
			TodayWorkflows: dto.DashboardWorkflowTable{Items: []dto.DashboardWorkflowRow{}},
			ScheduledRuns:  dto.DashboardScheduledRunTable{Items: []dto.DashboardScheduledRunRow{}},
			FailedRuns:     dto.DashboardWorkflowTable{Items: []dto.DashboardWorkflowRow{}},
			NodeActivity:   dto.DashboardNodeActivityTable{Items: []dto.DashboardNodeActivityRow{}},
		},
	}, nil
}

func TestDashboardHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	provider := &stubDashboardProvider{}
	handler := NewDashboardHandler(provider)
	router := gin.New()
	router.GET("/api/dashboard", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard?range=7d&tz=Asia/Shanghai", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "7d", provider.rangeKey)
	assert.Equal(t, "Asia/Shanghai", provider.timezone)

	var response APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response.Success)
}

func TestDashboardHandler_Get_DefaultsAndFallbacks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	provider := &stubDashboardProvider{}
	handler := NewDashboardHandler(provider)
	router := gin.New()
	router.GET("/api/dashboard", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard?tz=Invalid/Zone", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "30d", provider.rangeKey)
	assert.Equal(t, "UTC", provider.timezone)
}

func TestDashboardHandler_Get_InvalidRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewDashboardHandler(&stubDashboardProvider{})
	router := gin.New()
	router.GET("/api/dashboard", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard?range=14d", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, "range must be one of 7d, 30d, 90d", response.Error.Message)
}
