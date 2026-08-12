// 本文件验证完整路由树能够无冲突地完成注册。
package router

import (
	"io"
	"log/slog"
	"testing"

	"github.com/tokenpulse/tokenpulse/server/internal/config"
	"github.com/tokenpulse/tokenpulse/server/internal/handler"
)

// TestRoutesDoNotConflict 防止静态路由和参数路由发生 Gin 注册冲突。
func TestRoutesDoNotConflict(t *testing.T) {
	cfg := config.Config{Env: "test", JWTSecret: "test", CORSAllowedOrigins: []string{"http://localhost"}}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("route registration panicked: %v", recovered)
		}
	}()
	engine, err := New(cfg, handler.New(cfg, nil), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if len(engine.Routes()) == 0 {
		t.Fatal("no routes were registered")
	}
	foundDayDetail := false
	for _, route := range engine.Routes() {
		if route.Method == "GET" && route.Path == "/api/v1/statistics/day-detail" {
			foundDayDetail = true
			break
		}
	}
	if !foundDayDetail {
		t.Fatal("statistics day detail route was not registered")
	}
}
