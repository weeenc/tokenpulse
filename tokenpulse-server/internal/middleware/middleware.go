// Package middleware 提供日志、安全、认证、限流和请求资源控制等 Gin 中间件。
package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tokenpulse/tokenpulse/server/internal/api"
	"github.com/tokenpulse/tokenpulse/server/internal/security"
	"github.com/tokenpulse/tokenpulse/server/internal/service"
)

// Gin 上下文键集中定义在此处，供认证中间件、Handler 和日志中间件共享。
const (
	UserIDKey         = "userId"         // UserIDKey 是用户认证结果在 Gin 上下文中的键。
	DeviceIdentityKey = "deviceIdentity" // DeviceIdentityKey 是设备认证结果在 Gin 上下文中的键。
)

// RequestLogger 记录请求 ID、路由、状态码、耗时以及已认证主体。
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 复用上游请求 ID 便于链路追踪；缺失时由服务端生成。
			requestID = randomID()
		}
		c.Header("X-Request-ID", requestID)
		start := time.Now()
		c.Next()
		attrs := []any{"requestId", requestID, "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "latencyMs", time.Since(start).Milliseconds()}
		if userID, ok := c.Get(UserIDKey); ok {
			attrs = append(attrs, "userId", userID)
		}
		if raw, ok := c.Get(DeviceIdentityKey); ok {
			if identity, valid := raw.(service.DeviceIdentity); valid {
				attrs = append(attrs, "deviceId", identity.DeviceID)
			}
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "error", c.Errors.Last().Error())
		}
		logger.Info("http request", attrs...)
	}
}

// BodyLimit 限制请求体字节数，避免超大负载占用过多内存或网络资源。
func BodyLimit(bytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, bytes)
		c.Next()
	}
}

// ContextTimeout 为每个请求注入统一截止时间，并向数据库调用传播取消信号。
func ContextTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// CORS 仅允许白名单浏览器来源，并正确响应预检请求。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		_, originAllowed := allowed[origin]
		if origin != "" && !originAllowed && !sameOrigin(c, origin) {
			api.Error(c, http.StatusForbidden, 40302, "origin is not allowed")
			return
		}
		if originAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization,X-Request-ID,X-CSRF-Token")
			c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// sameOrigin 判断浏览器来源是否与当前反向代理入口一致；同源请求无需进入跨域白名单。
func sameOrigin(c *gin.Context, rawOrigin string) bool {
	origin, err := url.Parse(rawOrigin)
	if err != nil || origin.Scheme == "" || origin.Host == "" || origin.Path != "" {
		return false
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]); forwarded != "" {
		scheme = forwarded
	}
	requestOrigin, err := url.Parse(scheme + "://" + c.Request.Host)
	if err != nil {
		return false
	}
	return strings.EqualFold(origin.Scheme, requestOrigin.Scheme) &&
		strings.EqualFold(origin.Hostname(), requestOrigin.Hostname()) &&
		originPort(origin) == originPort(requestOrigin)
}

func originPort(value *url.URL) string {
	if value.Port() != "" {
		return value.Port()
	}
	if strings.EqualFold(value.Scheme, "https") {
		return "443"
	}
	return "80"
}

// CSRF 使用 Double Submit Cookie 模式比较 Cookie 与请求头中的随机令牌。
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("tp_csrf")
		header := c.GetHeader("X-CSRF-Token")
		if err != nil || cookie == "" || header == "" || len(cookie) != len(header) || subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) != 1 {
			api.Error(c, http.StatusForbidden, 40303, "invalid CSRF token")
			return
		}
		c.Next()
	}
}

// UserAuth 从 HttpOnly Cookie 或 Bearer 请求头验证用户访问令牌。
func UserAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("tp_access")
		if err != nil {
			header := c.GetHeader("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				token = strings.TrimPrefix(header, "Bearer ")
			}
		}
		claims, err := security.ParseJWT(token, "access", secret)
		if err != nil {
			api.Error(c, http.StatusUnauthorized, 40101, "authentication required")
			return
		}
		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

// DeviceAuth 验证 Agent 的设备令牌并把设备身份写入请求上下文。
func DeviceAuth(devices *service.DeviceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer dt_") {
			api.Error(c, http.StatusUnauthorized, 40102, "invalid device authorization")
			return
		}
		identity, err := devices.WithContext(c.Request.Context()).Authenticate(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			api.Error(c, http.StatusUnauthorized, 40102, "device authorization expired or revoked")
			return
		}
		c.Set(DeviceIdentityKey, *identity)
		c.Next()
	}
}

// rateEntry 保存单个限流键在当前窗口内的计数和重置时刻。
type rateEntry struct {
	count int       // count 为当前窗口内已经到达的请求数。
	reset time.Time // reset 为窗口失效时刻。
}

// RateLimiter 是按“客户端 IP + Gin 路由”计数的进程内固定窗口限流器。
type RateLimiter struct {
	mu          sync.Mutex           // mu 保护 entries 和 lastCleanup 的并发访问。
	entries     map[string]rateEntry // entries 保存各限流键的窗口状态。
	limit       int                  // limit 为单个窗口允许的最大请求数。
	window      time.Duration        // window 为固定窗口时长。
	lastCleanup time.Time            // lastCleanup 记录上次清理过期键的时间。
}

// NewRateLimiter 创建指定阈值和窗口长度的限流器。
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{entries: make(map[string]rateEntry), limit: limit, window: window, lastCleanup: time.Now()}
}

// Middleware 返回执行计数、周期清理和超限拒绝的 Gin 中间件。
func (l *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP() + ":" + c.FullPath()
		now := time.Now()
		l.mu.Lock()
		// 每个窗口最多进行一次全表清理，限制长期运行时的内存增长。
		if now.Sub(l.lastCleanup) >= l.window {
			for existingKey, existing := range l.entries {
				if !existing.reset.After(now) {
					delete(l.entries, existingKey)
				}
			}
			l.lastCleanup = now
		}
		entry := l.entries[key]
		if now.After(entry.reset) {
			entry = rateEntry{reset: now.Add(l.window)}
		}
		entry.count++
		l.entries[key] = entry
		allowed := entry.count <= l.limit
		l.mu.Unlock()
		if !allowed {
			api.Error(c, http.StatusTooManyRequests, 42901, "too many requests")
			return
		}
		c.Next()
	}
}

// UserID 读取用户认证中间件写入的用户 ID；缺失或类型错误时返回 0。
func UserID(c *gin.Context) uint64 { value, _ := c.Get(UserIDKey); id, _ := value.(uint64); return id }

// DeviceIdentity 读取设备认证中间件写入的完整设备身份。
func DeviceIdentity(c *gin.Context) service.DeviceIdentity {
	value, _ := c.Get(DeviceIdentityKey)
	identity, _ := value.(service.DeviceIdentity)
	return identity
}

// randomID 生成 96 位随机请求 ID；系统随机源失败时退化为高精度 UTC 时间戳。
func randomID() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return time.Now().UTC().Format("20060102150405.000000")
	}
	return hex.EncodeToString(buffer)
}
