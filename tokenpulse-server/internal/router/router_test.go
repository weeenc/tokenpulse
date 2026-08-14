// 本文件验证完整路由树能够无冲突地完成注册。
package router

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestLogoutWithoutCSRF 验证 CSRF Cookie 丢失时仍可退出并清理浏览器认证 Cookie。
func TestLogoutWithoutCSRF(t *testing.T) {
	cfg := config.Config{Env: "test", JWTSecret: "test", CORSAllowedOrigins: []string{"http://localhost"}}
	engine, err := New(cfg, handler.New(cfg, nil), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("logout without CSRF returned status %d: %s", response.Code, response.Body.String())
	}
	setCookies := strings.Join(response.Header().Values("Set-Cookie"), "\n")
	for _, name := range []string{"tp_access", "tp_refresh", "tp_csrf"} {
		if !strings.Contains(setCookies, name+"=") || !strings.Contains(setCookies, name+"=; Path=") {
			t.Errorf("logout did not clear %s cookie: %s", name, setCookies)
		}
	}
}

// TestLogoutRejectsCrossOriginRequest 确保退出免除双提交令牌后仍受全局 Origin 校验保护。
func TestLogoutRejectsCrossOriginRequest(t *testing.T) {
	cfg := config.Config{Env: "test", JWTSecret: "test", CORSAllowedOrigins: []string{"http://localhost"}}
	engine, err := New(cfg, handler.New(cfg, nil), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-origin logout returned status %d, want %d", response.Code, http.StatusForbidden)
	}
	if len(response.Header().Values("Set-Cookie")) != 0 {
		t.Fatal("cross-origin logout unexpectedly cleared authentication cookies")
	}
}
