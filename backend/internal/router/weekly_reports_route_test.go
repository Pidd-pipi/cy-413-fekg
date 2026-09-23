package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blueship581/mindgarden/backend/internal/config"
	"github.com/blueship581/mindgarden/backend/internal/handler"
	"github.com/blueship581/mindgarden/backend/internal/service"
	"io"
	"log/slog"
)

// TestWeeklyReportRoutesRegisterAndDispatch 确保 /weekly-reports/latest 静态路由
// 与 /weekly-reports/:id 参数路由可以共存并分别命中，且鉴权中间件仍生效。
func TestWeeklyReportRoutesRegisterAndDispatch(t *testing.T) {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	// service 传 nil 仓储即可：未带 JWT 时请求应在鉴权层被拒，不会触达 service。
	h := Handlers{WeeklyReport: handler.NewWeeklyReportHandler((*service.WeeklyReportService)(nil), l)}
	r := New(config.Config{JWTSecret: "secret", JWTIssuer: "t", CORSOrigin: "*"}, h, l)

	for _, path := range []string{"/api/v1/weekly-reports/latest", "/api/v1/weekly-reports/1", "/api/v1/weekly-reports", "/api/v1/weekly-reports/generate"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusUnauthorized {
			continue
		}
		t.Fatalf("GET %s without token: code=%d body=%s want 401", path, w.Code, strings.TrimSpace(w.Body.String()))
	}
}
