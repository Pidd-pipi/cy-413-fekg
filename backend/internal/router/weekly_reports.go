package router

import (
	"github.com/blueship581/mindgarden/backend/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterWeeklyReports(g *gin.RouterGroup, h *handler.WeeklyReportHandler, auth gin.HandlerFunc) {
	p := g.Group("/weekly-reports", auth)
	p.GET("/current", h.Current)
	p.POST("/generate", h.Generate)
}
