package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/middleware"
	"github.com/blueship581/mindgarden/backend/internal/service"
	"github.com/blueship581/mindgarden/backend/internal/util"
	"github.com/gin-gonic/gin"
)

type WeeklyReportHandler struct {
	s      *service.WeeklyReportService
	logger *slog.Logger
}

func NewWeeklyReportHandler(s *service.WeeklyReportService, l *slog.Logger) *WeeklyReportHandler {
	return &WeeklyReportHandler{s, l}
}

// Current 打开某结束日（默认今天）的当前周报快照。
func (h *WeeklyReportHandler) Current(c *gin.Context) {
	v, e := h.s.Get(middleware.UserID(c), c.Query("end_date"))
	if e != nil {
		c.Error(e)
		return
	}
	h.logger.Info(constants.LogWeeklyReportRead)
	ok(c, v)
}

// Generate 生成当前周报；同一账号同一结束日重复生成返回已有快照且不覆盖。
// 请求体允许为空（结束日默认今天）。
func (h *WeeklyReportHandler) Generate(c *gin.Context) {
	var r dto.WeeklyReportGenerateRequest
	raw, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil {
		c.Error(util.NewAppError(constants.CodeValidation, "Request[body] validation failed: malformed JSON", readErr))
		return
	}
	if len(strings.TrimSpace(string(raw))) > 0 {
		if e := json.Unmarshal(raw, &r); e != nil {
			c.Error(util.NewAppError(constants.CodeValidation, "Request[body] validation failed: malformed JSON", e))
			return
		}
		if r.EndDate != "" {
			if _, e := time.Parse(constants.WeeklyReportDateLayout, r.EndDate); e != nil {
				c.Error(util.NewAppError(constants.CodeValidation, "WeeklyReport[end_date] generate failed: invalid date", e))
				return
			}
		}
	}
	v, e := h.s.Generate(middleware.UserID(c), r.EndDate)
	if e != nil {
		c.Error(e)
		return
	}
	if v.AlreadyExist {
		ok(c, v)
		return
	}
	created(c, v)
}
