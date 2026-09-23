package handler

import (
	"strconv"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/middleware"
	"github.com/blueship581/mindgarden/backend/internal/service"
	"github.com/blueship581/mindgarden/backend/internal/util"
	"github.com/gin-gonic/gin"
	"log/slog"
)

type WeeklyReportHandler struct {
	s      *service.WeeklyReportService
	logger *slog.Logger
}

func NewWeeklyReportHandler(s *service.WeeklyReportService, l *slog.Logger) *WeeklyReportHandler {
	return &WeeklyReportHandler{s, l}
}

// Generate POST /weekly-reports/generate
func (h *WeeklyReportHandler) Generate(c *gin.Context) {
	var r dto.WeeklyReportGenerateRequest
	if c.Request.ContentLength > 0 {
		if !bind(c, &r) {
			return
		}
	}
	v, e := h.s.Generate(middleware.UserID(c), r.EndDate)
	if e != nil {
		c.Error(e)
		return
	}
	h.logger.Info(constants.LogWeeklyReportGenerated, "user_id", middleware.UserID(c), "reused", v.Reused)
	created(c, v)
}

// Latest GET /weekly-reports/latest
func (h *WeeklyReportHandler) Latest(c *gin.Context) {
	v, e := h.s.Latest(middleware.UserID(c))
	if e != nil {
		c.Error(e)
		return
	}
	ok(c, v)
}

// Get GET /weekly-reports/:id
func (h *WeeklyReportHandler) Get(c *gin.Context) {
	id, e := strconv.ParseUint(c.Param("id"), 10, 64)
	if e != nil {
		c.Error(util.NewAppError(constants.CodeValidation, "WeeklyReport[id] read failed: invalid id", e))
		return
	}
	v, e := h.s.Get(middleware.UserID(c), uint(id))
	if e != nil {
		c.Error(e)
		return
	}
	ok(c, v)
}

// GetByEndDate GET /weekly-reports?end_date=YYYY-MM-DD（缺省时返回最新一期）
func (h *WeeklyReportHandler) GetByEndDate(c *gin.Context) {
	endDate := c.Query("end_date")
	if endDate == "" {
		h.Latest(c)
		return
	}
	v, e := h.s.GetByEndDate(middleware.UserID(c), endDate)
	if e != nil {
		c.Error(e)
		return
	}
	ok(c, v)
}
