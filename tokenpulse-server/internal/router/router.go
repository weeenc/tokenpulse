// Package router 负责组装 Gin 中间件和 TokenPulse API 路由。
package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tokenpulse/tokenpulse/server/internal/config"
	"github.com/tokenpulse/tokenpulse/server/internal/handler"
	"github.com/tokenpulse/tokenpulse/server/internal/middleware"
)

// New 创建完整的 HTTP 路由树，并校验受信任代理配置。
func New(cfg config.Config, h *handler.Handler, logger *slog.Logger) (*gin.Engine, error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery(), middleware.RequestLogger(logger), middleware.ContextTimeout(15*time.Second), middleware.BodyLimit(2<<20))
	// 全局中间件按恢复、日志、超时、请求体限制和 CORS 的顺序执行。
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.GET("/health", h.Health)
	v1 := r.Group("/api/v1")
	// 登录和设备码轮询属于匿名高风险接口，按“客户端 IP + 路由”限流。
	limiter := middleware.NewRateLimiter(30, time.Minute)
	auth := v1.Group("/auth")
	auth.POST("/register", limiter.Middleware(), h.Register)
	auth.POST("/login", limiter.Middleware(), h.Login)
	auth.POST("/refresh", limiter.Middleware(), middleware.CSRF(), h.Refresh)
	// 退出必须保持幂等，即使 CSRF Cookie 已过期或缺失也要能清理认证 Cookie。
	// 浏览器跨站请求仍由全局 Origin 校验和 SameSite=Lax Cookie 策略阻断。
	auth.POST("/logout", h.Logout)
	auth.GET("/me", middleware.UserAuth(cfg.JWTSecret), h.Me)
	deviceAuth := v1.Group("/device-auth")
	deviceAuth.POST("/request", limiter.Middleware(), h.DeviceAuthRequest)
	deviceAuth.POST("/token", limiter.Middleware(), h.DeviceAuthToken)
	deviceAuth.GET("/info/:userCode", middleware.UserAuth(cfg.JWTSecret), h.DeviceAuthInfo)
	deviceAuth.POST("/approve", middleware.UserAuth(cfg.JWTSecret), middleware.CSRF(), h.DeviceAuthApprove)
	deviceAuth.POST("/deny", middleware.UserAuth(cfg.JWTSecret), middleware.CSRF(), h.DeviceAuthDeny)
	users := v1.Group("")
	// 用户接口依赖短期访问令牌，并对所有状态变更操作额外校验 CSRF。
	users.Use(middleware.UserAuth(cfg.JWTSecret))
	users.GET("/devices", h.Devices)
	users.GET("/devices/:id", h.Device)
	users.PATCH("/devices/:id", middleware.CSRF(), h.RenameDevice)
	users.POST("/devices/:id/revoke", middleware.CSRF(), h.RevokeDevice)
	users.GET("/statistics/summary", h.StatisticsSummary)
	users.GET("/statistics/trend", h.StatisticsTrend)
	users.GET("/statistics/day-detail", h.StatisticsDayDetail)
	users.GET("/statistics/by-device", h.StatisticsBy("device"))
	users.GET("/statistics/by-source", h.StatisticsBy("source"))
	users.GET("/statistics/by-model", h.StatisticsBy("model"))
	users.GET("/statistics/recent", h.StatisticsRecent)
	device := v1.Group("")
	// Agent 接口使用独立的设备凭据，不复用浏览器 Cookie 会话。
	device.Use(middleware.DeviceAuth(h.DeviceService()))
	device.GET("/devices/me", h.DeviceMe)
	device.POST("/devices/heartbeat", h.Heartbeat)
	device.GET("/agent/config", h.AgentConfig)
	device.POST("/usage/batch", h.UsageBatch)
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": 40400, "message": "not found", "data": nil})
	})
	return r, nil
}
