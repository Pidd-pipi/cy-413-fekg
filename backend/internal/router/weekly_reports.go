package router

import (
	"github.com/blueship581/mindgarden/backend/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterWeeklyReports(g *gin.RouterGroup, h *handler.WeeklyReportHandler, auth gin.HandlerFunc) {
	p := g.Group("/weekly-reports", auth)
	p.POST("/generate", h.Generate)
	p.GET("/latest", h.Latest)
	p.GET("", h.GetByEndDate)
	p.GET("/:id", h.Get)
}
