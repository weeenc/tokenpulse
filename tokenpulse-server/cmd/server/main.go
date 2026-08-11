// Package main 提供 TokenPulse HTTP 服务的进程入口。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tokenpulse/tokenpulse/server/internal/config"
	"github.com/tokenpulse/tokenpulse/server/internal/database"
	"github.com/tokenpulse/tokenpulse/server/internal/handler"
	"github.com/tokenpulse/tokenpulse/server/internal/router"
)

// main 完成启动前依赖检查，启动后台维护与 HTTP 服务，并处理优雅退出。
func main() {
	// 统一使用结构化 JSON 日志，便于容器平台采集和检索。
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// 配置、数据库和迁移必须在监听端口前全部就绪，避免服务带病接收流量。
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}
	db, err := database.Open(cfg.MySQLDSN, cfg.Env == "production")
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	if err := database.ApplyMigrations(db, cfg.MigrationsDir); err != nil {
		logger.Error("apply database migrations", "error", err)
		os.Exit(1)
	}
	h := handler.New(cfg, db)
	engine, err := router.New(cfg, h, logger)
	if err != nil {
		logger.Error("configure router", "error", err)
		os.Exit(1)
	}
	server := &http.Server{Addr: ":" + cfg.Port, Handler: engine, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	// 后台维护任务和 HTTP 监听互不阻塞。
	go runMaintenance(logger, h)
	go func() {
		logger.Info("server started", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	// 收到退出信号后给正在处理的请求最多 15 秒完成时间。
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown", "error", err)
	}
}

// runMaintenance 周期性清理过期的设备授权请求和刷新会话。
func runMaintenance(logger *slog.Logger, h *handler.Handler) {
	deviceTicker := time.NewTicker(time.Minute)
	refreshTicker := time.NewTicker(24 * time.Hour)
	defer deviceTicker.Stop()
	defer refreshTicker.Stop()
	for {
		select {
		case <-deviceTicker.C:
			if err := h.DeviceAuthService().ExpirePending(); err != nil {
				logger.Error("expire device requests", "error", err)
			}
		case <-refreshTicker.C:
			if err := h.AuthService().DeleteExpiredRefreshSessions(7 * 24 * time.Hour); err != nil {
				logger.Error("delete expired refresh sessions", "error", err)
			}
		}
	}
}
