// Package config 负责从环境变量加载并校验服务端运行配置。
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 汇总服务启动和各业务模块需要的配置项。
type Config struct {
	Env                string        // Env 为运行环境，例如 development、test 或 production。
	Port               string        // Port 为 HTTP 服务监听端口。
	MySQLDSN           string        // MySQLDSN 为完整的 MySQL 连接字符串。
	JWTSecret          string        // JWTSecret 用于签发和验证用户访问令牌。
	AccessExpire       time.Duration // AccessExpire 为访问令牌有效期。
	RefreshExpire      time.Duration // RefreshExpire 为刷新会话有效期。
	WebBaseURL         string        // WebBaseURL 为用户完成设备授权的 Web 站点地址。
	DeviceAuthExpire   time.Duration // DeviceAuthExpire 为设备授权码有效期。
	CORSAllowedOrigins []string      // CORSAllowedOrigins 为允许浏览器跨域访问的来源白名单。
	TrustedProxies     []string      // TrustedProxies 为 Gin 可信任的反向代理地址或网段。
	MigrationsDir      string        // MigrationsDir 为版本化 SQL 迁移文件目录。
}

// Load 读取环境变量、应用安全默认值并完成生产环境约束校验。
func Load() (Config, error) {
	c := Config{
		Env:                env("APP_ENV", "development"),
		Port:               env("SERVER_PORT", "8080"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		WebBaseURL:         env("WEB_BASE_URL", "http://localhost:5173"),
		CORSAllowedOrigins: splitCSV(env("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		TrustedProxies:     splitCSV(os.Getenv("TRUSTED_PROXIES")),
		MigrationsDir:      env("MIGRATIONS_DIR", "migrations"),
	}
	if c.JWTSecret == "" {
		if c.Env != "test" {
			return Config{}, fmt.Errorf("JWT_SECRET is required")
		}
		c.JWTSecret = "test-only-secret"
	}
	if c.Env == "production" && len(c.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters in production")
	}
	if err := validatePublicURLs(c); err != nil {
		return Config{}, err
	}
	var err error
	if c.AccessExpire, err = duration("JWT_ACCESS_EXPIRE", 15*time.Minute); err != nil {
		return Config{}, err
	}
	if c.RefreshExpire, err = duration("JWT_REFRESH_EXPIRE", 30*24*time.Hour); err != nil {
		return Config{}, err
	}
	if c.DeviceAuthExpire, err = duration("DEVICE_AUTH_EXPIRE", 10*time.Minute); err != nil {
		return Config{}, err
	}
	if dsn := os.Getenv("MYSQL_DSN"); dsn != "" {
		// 显式 DSN 优先，便于云数据库或带额外参数的部署环境使用。
		c.MySQLDSN = dsn
	} else {
		if c.Env == "production" && os.Getenv("MYSQL_PASSWORD") == "" {
			return Config{}, fmt.Errorf("MYSQL_PASSWORD is required in production")
		}
		c.MySQLDSN = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			env("MYSQL_USERNAME", "tokenpulse"), env("MYSQL_PASSWORD", "tokenpulse"),
			env("MYSQL_HOST", "127.0.0.1"), env("MYSQL_PORT", "3306"), env("MYSQL_DATABASE", "token_usage"))
	}
	return c, nil
}

// validatePublicURLs 校验公开地址及跨域来源，生产环境强制使用 HTTPS。
func validatePublicURLs(c Config) error {
	web, err := url.Parse(c.WebBaseURL)
	if err != nil || web.Scheme == "" || web.Host == "" {
		return fmt.Errorf("WEB_BASE_URL must be an absolute URL")
	}
	if c.Env == "production" && web.Scheme != "https" {
		return fmt.Errorf("WEB_BASE_URL must use HTTPS in production")
	}
	if len(c.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must not be empty")
	}
	for _, raw := range c.CORSAllowedOrigins {
		origin, parseErr := url.Parse(raw)
		if parseErr != nil || origin.Scheme == "" || origin.Host == "" || origin.Path != "" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS contains an invalid origin")
		}
		if c.Env == "production" && origin.Scheme != "https" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS must use HTTPS in production")
		}
	}
	return nil
}

// env 返回非空环境变量；未配置时使用给定默认值。
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// duration 同时支持纯秒数和 Go duration（如 15m、24h）两种写法。
func duration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}

// splitCSV 将逗号分隔配置拆成去空白、去空项的字符串切片。
func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
