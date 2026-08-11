// 本文件覆盖安全与资源控制中间件的关键行为。
package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestCSRFMiddleware 验证匹配令牌通过、不同令牌被拒绝。
func TestCSRFMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/write", CSRF(), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	valid := httptest.NewRequest(http.MethodPost, "/write", nil)
	valid.AddCookie(&http.Cookie{Name: "tp_csrf", Value: "secret"})
	valid.Header.Set("X-CSRF-Token", "secret")
	validResponse := httptest.NewRecorder()
	engine.ServeHTTP(validResponse, valid)
	if validResponse.Code != http.StatusNoContent {
		t.Fatalf("valid token returned %d", validResponse.Code)
	}

	invalid := httptest.NewRequest(http.MethodPost, "/write", nil)
	invalid.AddCookie(&http.Cookie{Name: "tp_csrf", Value: "secret"})
	invalid.Header.Set("X-CSRF-Token", "different")
	invalidResponse := httptest.NewRecorder()
	engine.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusForbidden {
		t.Fatalf("invalid token returned %d", invalidResponse.Code)
	}
}

// TestCORSRejectsUntrustedBrowserOrigin 验证来源白名单不会放行恶意站点。
func TestCORSRejectsUntrustedBrowserOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(CORS([]string{"https://tokenpulse.example.com"}))
	engine.POST("/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("untrusted origin returned %d", response.Code)
	}
}

// TestRateLimiterRemovesExpiredEntries 验证限流器会回收过期键。
func TestRateLimiterRemovesExpiredEntries(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)
	limiter.entries["expired"] = rateEntry{count: 1, reset: time.Now().Add(-time.Minute)}
	limiter.lastCleanup = time.Now().Add(-2 * time.Minute)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/limited", limiter.Middleware(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/limited", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if _, exists := limiter.entries["expired"]; exists {
		t.Fatal("expired limiter entry was not removed")
	}
}

// TestContextTimeoutPropagatesDeadline 验证超时截止时间能传入下游 Handler。
func TestContextTimeoutPropagatesDeadline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/timeout", ContextTimeout(time.Millisecond), func(c *gin.Context) {
		<-c.Request.Context().Done()
		if !errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
			t.Fatalf("unexpected context error: %v", c.Request.Context().Err())
		}
		c.Status(http.StatusGatewayTimeout)
	})
	request := httptest.NewRequest(http.MethodGet, "/timeout", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("timeout returned %d", response.Code)
	}
}
