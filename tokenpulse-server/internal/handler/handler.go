// Package handler 负责 HTTP 参数校验、业务服务编排和响应错误映射。
package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tokenpulse/tokenpulse/server/internal/api"
	"github.com/tokenpulse/tokenpulse/server/internal/config"
	"github.com/tokenpulse/tokenpulse/server/internal/middleware"
	"github.com/tokenpulse/tokenpulse/server/internal/security"
	"github.com/tokenpulse/tokenpulse/server/internal/service"
	"gorm.io/gorm"
)

// Handler 聚合路由处理函数所需的配置、数据库和各领域服务。
type Handler struct {
	cfg        config.Config              // cfg 保存令牌、Cookie 和公开地址配置。
	db         *gorm.DB                   // db 用于健康检查。
	auth       *service.AuthService       // auth 处理账户与刷新会话。
	deviceAuth *service.DeviceAuthService // deviceAuth 处理设备授权状态机。
	devices    *service.DeviceService     // devices 处理设备凭据和设备管理。
	usage      *service.UsageService      // usage 处理用量批量入库。
	statistics *service.StatisticsService // statistics 处理统计查询。
}

// New 创建 Handler 并初始化全部无状态业务服务。
func New(cfg config.Config, db *gorm.DB) *Handler {
	return &Handler{cfg: cfg, db: db, auth: service.NewAuthService(db), deviceAuth: service.NewDeviceAuthService(db, cfg.WebBaseURL, cfg.DeviceAuthExpire), devices: service.NewDeviceService(db), usage: service.NewUsageService(db), statistics: service.NewStatisticsService(db)}
}

// DeviceService 暴露设备服务供设备认证中间件使用。
func (h *Handler) DeviceService() *service.DeviceService { return h.devices }

// DeviceAuthService 暴露设备授权服务供后台过期任务使用。
func (h *Handler) DeviceAuthService() *service.DeviceAuthService { return h.deviceAuth }

// AuthService 暴露认证服务供后台刷新会话清理任务使用。
func (h *Handler) AuthService() *service.AuthService { return h.auth }

// registerRequest 定义注册接口的 JSON 输入及声明式校验规则。
type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`  // Username 为登录用户名。
	Email    string `json:"email" binding:"omitempty,email,max=128"`   // Email 为可选登录邮箱。
	Password string `json:"password" binding:"required,min=8,max=128"` // Password 为待哈希的明文密码。
}

// Register 注册账户、创建刷新会话并设置浏览器认证 Cookie。
func (h *Handler) Register(c *gin.Context) {
	var request registerRequest
	if !bind(c, &request) {
		return
	}
	user, err := h.auth.WithContext(c.Request.Context()).Register(request.Username, request.Email, request.Password)
	if errors.Is(err, service.ErrConflict) {
		api.Error(c, http.StatusConflict, 40901, "username or email already exists")
		return
	}
	if err != nil {
		internal(c, err)
		return
	}
	if err := h.startSession(c, user.ID); err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusCreated, user)
}

// loginRequest 定义用户名/邮箱登录输入。
type loginRequest struct {
	Identity string `json:"identity" binding:"required,max=128"` // Identity 为用户名或邮箱。
	Password string `json:"password" binding:"required,max=128"` // Password 为待验证的明文密码。
}

// Login 验证账户凭据并启动新的浏览器会话。
func (h *Handler) Login(c *gin.Context) {
	var request loginRequest
	if !bind(c, &request) {
		return
	}
	user, err := h.auth.WithContext(c.Request.Context()).Login(request.Identity, request.Password)
	if errors.Is(err, service.ErrUnauthorized) {
		api.Error(c, http.StatusUnauthorized, 40103, "invalid username or password")
		return
	}
	if err != nil {
		internal(c, err)
		return
	}
	if err := h.startSession(c, user.ID); err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, user)
}

// Refresh 原子轮换刷新令牌，并重发访问令牌和 CSRF Cookie。
func (h *Handler) Refresh(c *gin.Context) {
	token, err := c.Cookie("tp_refresh")
	if err != nil {
		api.Error(c, http.StatusUnauthorized, 40104, "refresh token required")
		return
	}
	userID, replacement, err := h.auth.WithContext(c.Request.Context()).RotateRefreshSession(token, h.cfg.RefreshExpire)
	if err != nil {
		api.Error(c, http.StatusUnauthorized, 40104, "invalid refresh token")
		return
	}
	if err := h.setTokens(c, userID, replacement); err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, map[string]bool{"refreshed": true})
}

// Logout 撤销当前刷新令牌并删除全部浏览器认证 Cookie。
func (h *Handler) Logout(c *gin.Context) {
	if token, err := c.Cookie("tp_refresh"); err == nil {
		if err := h.auth.WithContext(c.Request.Context()).RevokeRefreshSession(token); err != nil {
			internal(c, err)
			return
		}
	}
	h.clearTokens(c)
	api.OK(c, http.StatusOK, map[string]bool{"loggedOut": true})
}

// Me 返回当前已认证用户的公开账户信息。
func (h *Handler) Me(c *gin.Context) {
	user, err := h.auth.WithContext(c.Request.Context()).User(middleware.UserID(c))
	if err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, user)
}

// deviceAuthRequest 定义 Agent 发起设备授权所需的安装信息。
type deviceAuthRequest struct {
	DeviceName     string `json:"deviceName" binding:"required,max=128"`   // DeviceName 为设备展示名称。
	Platform       string `json:"platform" binding:"required,max=32"`      // Platform 为操作系统。
	Arch           string `json:"arch" binding:"required,max=32"`          // Arch 为处理器架构。
	AgentVersion   string `json:"agentVersion" binding:"omitempty,max=32"` // AgentVersion 为客户端版本。
	InstallationID string `json:"installationId" binding:"required"`       // InstallationID 为安装 UUID。
}

// DeviceAuthRequest 创建设备授权请求并返回轮询与验证信息。
func (h *Handler) DeviceAuthRequest(c *gin.Context) {
	var request deviceAuthRequest
	if !bind(c, &request) {
		return
	}
	if _, err := uuid.Parse(request.InstallationID); err != nil {
		api.Error(c, http.StatusBadRequest, 40001, "installationId must be a UUID")
		return
	}
	created, deviceCode, err := h.deviceAuth.WithContext(c.Request.Context()).Request(service.DeviceAuthInput{DeviceName: request.DeviceName, Platform: request.Platform, Arch: request.Arch, AgentVersion: request.AgentVersion, InstallationUUID: request.InstallationID})
	if err != nil {
		internal(c, err)
		return
	}
	verification := strings.TrimRight(h.cfg.WebBaseURL, "/") + "/device"
	api.OK(c, http.StatusCreated, map[string]any{"deviceCode": deviceCode, "userCode": created.UserCode, "verificationUri": verification, "verificationUriComplete": verification + "?code=" + url.QueryEscape(created.UserCode), "expiresIn": int(h.cfg.DeviceAuthExpire.Seconds()), "interval": 5})
}

// deviceTokenRequest 定义 Agent 轮询授权状态使用的设备码。
type deviceTokenRequest struct {
	DeviceCode string `json:"deviceCode" binding:"required,max=128"` // DeviceCode 为申请阶段获得的高熵明文码。
}

// DeviceAuthToken 根据设备码返回等待状态、失败状态或一次性设备令牌。
func (h *Handler) DeviceAuthToken(c *gin.Context) {
	var request deviceTokenRequest
	if !bind(c, &request) {
		return
	}
	result, state, err := h.deviceAuth.WithContext(c.Request.Context()).Token(request.DeviceCode)
	if err != nil {
		status := http.StatusBadRequest
		code := 40002
		if errors.Is(err, service.ErrUnauthorized) {
			status, code, state = http.StatusUnauthorized, 40105, "invalid_device_code"
		}
		api.Error(c, status, code, state)
		return
	}
	api.OK(c, http.StatusOK, map[string]any{"deviceToken": result.DeviceToken, "account": result.Username, "device": map[string]any{"deviceId": result.Device.DeviceUUID, "deviceName": result.Device.DeviceName}})
}

// DeviceAuthInfo 返回用户码对应的待审批请求和用户设备列表。
func (h *Handler) DeviceAuthInfo(c *gin.Context) {
	request, devices, err := h.deviceAuth.WithContext(c.Request.Context()).Info(middleware.UserID(c), c.Param("userCode"))
	if errors.Is(err, service.ErrNotFound) {
		api.Error(c, http.StatusNotFound, 40401, "device request not found")
		return
	}
	if errors.Is(err, service.ErrExpired) {
		api.Error(c, http.StatusGone, 41001, "device authorization expired")
		return
	}
	if err != nil {
		api.Error(c, http.StatusConflict, 40902, "device request is no longer pending")
		return
	}
	api.OK(c, http.StatusOK, map[string]any{"request": map[string]any{"userCode": request.UserCode, "deviceName": request.DeviceName, "platform": request.Platform, "arch": request.Arch, "agentVersion": request.AgentVersion, "expiresAt": request.ExpiresAt}, "devices": devices})
}

// approveRequest 定义用户批准新设备或重新连接已有设备的输入。
type approveRequest struct {
	UserCode       string  `json:"userCode" binding:"required,max=16"` // UserCode 为用户看到的短授权码。
	TargetDeviceID *uint64 `json:"targetDeviceId"`                     // TargetDeviceID 非空时重新连接已有设备。
}

// DeviceAuthApprove 批准待处理设备授权请求。
func (h *Handler) DeviceAuthApprove(c *gin.Context) {
	var request approveRequest
	if !bind(c, &request) {
		return
	}
	err := h.deviceAuth.WithContext(c.Request.Context()).Approve(middleware.UserID(c), request.UserCode, request.TargetDeviceID)
	if errors.Is(err, service.ErrForbidden) {
		api.Error(c, http.StatusForbidden, 40301, "target device does not belong to this user")
		return
	}
	if errors.Is(err, service.ErrExpired) {
		api.Error(c, http.StatusGone, 41001, "device authorization expired")
		return
	}
	if errors.Is(err, service.ErrNotFound) {
		api.Error(c, http.StatusNotFound, 40401, "device request not found")
		return
	}
	if err != nil {
		api.Error(c, http.StatusConflict, 40902, "device request is no longer pending")
		return
	}
	api.OK(c, http.StatusOK, map[string]bool{"approved": true})
}

// denyRequest 定义拒绝设备授权的输入。
type denyRequest struct {
	UserCode string `json:"userCode" binding:"required,max=16"` // UserCode 为待拒绝请求的用户码。
}

// DeviceAuthDeny 拒绝待处理设备授权请求。
func (h *Handler) DeviceAuthDeny(c *gin.Context) {
	var request denyRequest
	if !bind(c, &request) {
		return
	}
	if err := h.deviceAuth.WithContext(c.Request.Context()).Deny(middleware.UserID(c), request.UserCode); err != nil {
		api.Error(c, http.StatusConflict, 40902, "device request is no longer pending")
		return
	}
	api.OK(c, http.StatusOK, map[string]bool{"denied": true})
}

// Devices 返回当前用户的设备列表及最新 Agent 版本。
func (h *Handler) Devices(c *gin.Context) {
	devices, err := h.devices.WithContext(c.Request.Context()).List(middleware.UserID(c))
	if err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, devices)
}

// Device 返回当前用户拥有的指定设备。
func (h *Handler) Device(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	device, err := h.devices.WithContext(c.Request.Context()).Get(middleware.UserID(c), id)
	if errors.Is(err, service.ErrNotFound) {
		api.Error(c, http.StatusNotFound, 40402, "device not found")
		return
	}
	if err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, device)
}

// renameRequest 定义设备重命名输入。
type renameRequest struct {
	DeviceName string `json:"deviceName" binding:"required,max=128"` // DeviceName 为新的展示名称。
}

// RenameDevice 重命名当前用户拥有的有效设备。
func (h *Handler) RenameDevice(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var request renameRequest
	if !bind(c, &request) {
		return
	}
	if err := h.devices.WithContext(c.Request.Context()).Rename(middleware.UserID(c), id, request.DeviceName); errors.Is(err, service.ErrNotFound) {
		api.Error(c, http.StatusNotFound, 40402, "device not found")
		return
	} else if err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, map[string]bool{"updated": true})
}

// RevokeDevice 撤销逻辑设备及其全部设备凭据。
func (h *Handler) RevokeDevice(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.devices.WithContext(c.Request.Context()).Revoke(middleware.UserID(c), id); errors.Is(err, service.ErrNotFound) {
		api.Error(c, http.StatusNotFound, 40402, "device not found")
		return
	} else if err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, map[string]bool{"revoked": true})
}

// DeviceMe 返回当前设备令牌对应的 Agent 配置信息和账户摘要。
func (h *Handler) DeviceMe(c *gin.Context) {
	data, err := h.devices.WithContext(c.Request.Context()).Me(middleware.DeviceIdentity(c))
	if err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, data)
}

// heartbeatRequest 定义 Agent 心跳携带的运行版本；字段可选以兼容旧客户端。
type heartbeatRequest struct {
	AgentVersion string `json:"agentVersion" binding:"omitempty,max=32"`
}

// Heartbeat 更新当前设备与安装的最近活动时间和 Agent 版本。
func (h *Handler) Heartbeat(c *gin.Context) {
	var request heartbeatRequest
	if !bind(c, &request) {
		return
	}
	if err := h.devices.WithContext(c.Request.Context()).Heartbeat(middleware.DeviceIdentity(c), request.AgentVersion); err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, map[string]any{"serverTime": time.Now().UTC()})
}

// AgentConfig 返回 Agent 批量上报大小和同步间隔等服务端策略。
func (h *Handler) AgentConfig(c *gin.Context) {
	api.OK(c, http.StatusOK, map[string]any{"maxBatchSize": 500, "syncIntervalSeconds": 3600})
}

// usageEventRequest 定义 Agent 上报的单条用量事件及边界校验。
type usageEventRequest struct {
	EventID           string    `json:"eventId" binding:"required,len=64,hexadecimal"` // EventID 为 64 位十六进制幂等标识。
	Source            string    `json:"source" binding:"required,max=64"`              // Source 为采集来源。
	Model             *string   `json:"model" binding:"omitempty,max=128"`             // Model 为可选模型名。
	SessionID         *string   `json:"sessionId" binding:"omitempty,max=255"`         // SessionID 为可选会话标识。
	MessageID         *string   `json:"messageId" binding:"omitempty,max=255"`         // MessageID 为可选消息标识。
	InputTokens       uint64    `json:"inputTokens"`                                   // InputTokens 为输入 Token 数。
	OutputTokens      uint64    `json:"outputTokens"`                                  // OutputTokens 为输出 Token 数。
	CachedInputTokens uint64    `json:"cachedInputTokens"`                             // CachedInputTokens 为缓存输入 Token 数。
	ReasoningTokens   uint64    `json:"reasoningTokens"`                               // ReasoningTokens 为推理 Token 数。
	TotalTokens       uint64    `json:"totalTokens"`                                   // TotalTokens 为来源上报的总数；部分来源可能没有输入/输出细分。
	OccurredAt        time.Time `json:"occurredAt" binding:"required"`                 // OccurredAt 为事件发生时间。
}

// usageBatchRequest 将单次同步限制在 1 到 500 条事件。
type usageBatchRequest struct {
	Events []usageEventRequest `json:"events" binding:"required,min=1,max=500,dive"` // Events 为逐条递归校验的事件集合。
}

// UsageBatch 校验 Token 数量关系、幂等写入批次并刷新设备心跳。
func (h *Handler) UsageBatch(c *gin.Context) {
	var request usageBatchRequest
	if !bind(c, &request) {
		return
	}
	inputs := make([]service.UsageInput, 0, len(request.Events))
	for _, event := range request.Events {
		// 通常总数应等于输入与输出之和；但部分 Codex 记录只提供经过验证的
		// total_tokens。仅在所有细分均为 0 时接受这种 total-only 记录，避免
		// 为了通过校验而伪造输入或输出用量。
		if !validTokenTotals(event) {
			api.Error(c, http.StatusBadRequest, 40003, "totalTokens must equal inputTokens + outputTokens unless token breakdown is unavailable")
			return
		}
		inputs = append(inputs, service.UsageInput{EventID: event.EventID, Source: event.Source, Model: event.Model, SessionID: event.SessionID, MessageID: event.MessageID, InputTokens: event.InputTokens, OutputTokens: event.OutputTokens, CachedInputTokens: event.CachedInputTokens, ReasoningTokens: event.ReasoningTokens, TotalTokens: event.TotalTokens, OccurredAt: event.OccurredAt})
	}
	inserted, err := h.usage.WithContext(c.Request.Context()).Batch(middleware.DeviceIdentity(c), inputs)
	if err != nil {
		internal(c, err)
		return
	}
	if err := h.devices.WithContext(c.Request.Context()).Heartbeat(middleware.DeviceIdentity(c), ""); err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, map[string]any{"received": len(inputs), "inserted": inserted, "duplicated": int64(len(inputs)) - inserted})
}

// validTokenTotals 接受完整且自洽的细分，或仅包含来源总量的记录。
func validTokenTotals(event usageEventRequest) bool {
	if event.TotalTokens == event.InputTokens+event.OutputTokens {
		return true
	}
	return event.TotalTokens > 0 && event.InputTokens == 0 && event.OutputTokens == 0 &&
		event.CachedInputTokens == 0 && event.ReasoningTokens == 0
}

// StatisticsSummary 返回当前用户的周期汇总和费用估算。
func (h *Handler) StatisticsSummary(c *gin.Context) {
	filter, ok := statisticsFilter(c)
	if !ok {
		return
	}
	result, err := h.statistics.WithContext(c.Request.Context()).Summary(middleware.UserID(c), filter)
	if err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, result)
}

// StatisticsTrend 返回按用户本地自然日聚合的用量趋势。
func (h *Handler) StatisticsTrend(c *gin.Context) {
	filter, ok := statisticsFilter(c)
	if !ok {
		return
	}
	result, err := h.statistics.WithContext(c.Request.Context()).Trend(middleware.UserID(c), filter)
	if err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, result)
}

// StatisticsBy 创建按设备、来源或模型维度聚合的处理函数。
func (h *Handler) StatisticsBy(group string) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, ok := statisticsFilter(c)
		if !ok {
			return
		}
		result, err := h.statistics.WithContext(c.Request.Context()).By(middleware.UserID(c), filter, group)
		if err != nil {
			internal(c, err)
			return
		}
		api.OK(c, http.StatusOK, result)
	}
}

// StatisticsRecent 返回当前筛选条件下最近发生的用量事件。
func (h *Handler) StatisticsRecent(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	filter, ok := statisticsFilter(c)
	if !ok {
		return
	}
	result, err := h.statistics.WithContext(c.Request.Context()).Recent(middleware.UserID(c), filter, limit)
	if err != nil {
		internal(c, err)
		return
	}
	api.OK(c, http.StatusOK, result)
}

// Health 检查数据库连接可用性，供负载均衡器和容器探针调用。
func (h *Handler) Health(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
		api.Error(c, http.StatusServiceUnavailable, 50301, "database unavailable")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// startSession 创建持久化刷新会话并设置完整 Cookie 集合。
func (h *Handler) startSession(c *gin.Context, userID uint64) error {
	refresh, err := h.auth.WithContext(c.Request.Context()).CreateRefreshSession(userID, h.cfg.RefreshExpire)
	if err != nil {
		return err
	}
	return h.setTokens(c, userID, refresh)
}

// setTokens 签发短期访问令牌、刷新 Cookie 和可读的 CSRF Cookie。
func (h *Handler) setTokens(c *gin.Context, userID uint64, refresh string) error {
	access, err := security.NewJWT(userID, "access", h.cfg.JWTSecret, h.cfg.AccessExpire)
	if err != nil {
		return err
	}
	csrf, err := security.RandomToken("", 24)
	if err != nil {
		return err
	}
	secure := requestIsSecure(c)
	// 认证 Cookie 使用 HttpOnly；CSRF Cookie 需要前端读取后回传请求头。
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("tp_access", access, int(h.cfg.AccessExpire.Seconds()), "/", "", secure, true)
	c.SetCookie("tp_refresh", refresh, int(h.cfg.RefreshExpire.Seconds()), "/api/v1/auth", "", secure, true)
	c.SetCookie("tp_csrf", csrf, int(h.cfg.RefreshExpire.Seconds()), "/", "", secure, false)
	return nil
}

// clearTokens 使用相同路径和安全属性把浏览器认证 Cookie 设为过期。
func (h *Handler) clearTokens(c *gin.Context) {
	secure := requestIsSecure(c)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("tp_access", "", -1, "/", "", secure, true)
	c.SetCookie("tp_refresh", "", -1, "/api/v1/auth", "", secure, true)
	c.SetCookie("tp_csrf", "", -1, "/", "", secure, false)
}

// requestIsSecure 根据反向代理传入的协议设置 Cookie 的 Secure 属性，兼容同一服务的 HTTP 与 HTTPS 入口。
func requestIsSecure(c *gin.Context) bool {
	return c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

// bind 统一执行 JSON 绑定与字段校验，并输出标准参数错误。
func bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		api.Error(c, http.StatusBadRequest, 40000, "invalid request: "+err.Error())
		return false
	}
	return true
}

// internal 把内部错误挂到 Gin 上下文供请求日志记录，对客户端隐藏细节。
func internal(c *gin.Context, err error) {
	_ = c.Error(err)
	api.Error(c, http.StatusInternalServerError, 50000, "internal server error")
}

// pathID 解析设备路由中的无符号整数主键。
func pathID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Error(c, http.StatusBadRequest, 40000, "invalid device id")
		return 0, false
	}
	return id, true
}

// statisticsFilter 解析并校验统计接口共享的时间、设备、来源、模型和时区参数。
func statisticsFilter(c *gin.Context) (service.StatisticsFilter, bool) {
	filter := service.StatisticsFilter{Source: c.Query("source"), Model: c.Query("model")}
	if len(filter.Source) > 64 || len(filter.Model) > 128 {
		api.Error(c, http.StatusBadRequest, 40004, "source or model filter is too long")
		return filter, false
	}
	if value := c.Query("startTime"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			api.Error(c, http.StatusBadRequest, 40004, "startTime must use RFC3339")
			return filter, false
		}
		filter.StartTime = &parsed
	}
	if value := c.Query("endTime"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			api.Error(c, http.StatusBadRequest, 40004, "endTime must use RFC3339")
			return filter, false
		}
		filter.EndTime = &parsed
	}
	if value := c.Query("deviceId"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			api.Error(c, http.StatusBadRequest, 40004, "deviceId must be an integer")
			return filter, false
		}
		filter.DeviceID = &parsed
	}
	if value := c.Query("timezoneOffsetMinutes"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < -840 || parsed > 840 {
			api.Error(c, http.StatusBadRequest, 40004, "timezoneOffsetMinutes is invalid")
			return filter, false
		}
		filter.TimezoneOffsetMinutes = parsed
	}
	if filter.StartTime != nil && filter.EndTime != nil && !filter.StartTime.Before(*filter.EndTime) {
		api.Error(c, http.StatusBadRequest, 40004, "startTime must be before endTime")
		return filter, false
	}
	return filter, true
}
