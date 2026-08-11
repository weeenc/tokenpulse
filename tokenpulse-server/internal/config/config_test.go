// 本文件验证配置加载的安全边界和环境变量解析行为。
package config

import "testing"

// TestLoadRequiresSecretOutsideTests 验证非测试环境禁止使用隐式 JWT 密钥。
func TestLoadRequiresSecretOutsideTests(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", "")
	if _, err := Load(); err == nil {
		t.Fatal("development must require an explicit JWT secret")
	}
}

// TestLoadParsesTrustedProxies 验证代理白名单的逗号分隔解析。
func TestLoadParsesTrustedProxies(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8, 192.168.1.4")
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.TrustedProxies) != 2 || loaded.TrustedProxies[1] != "192.168.1.4" {
		t.Fatalf("unexpected proxies: %#v", loaded.TrustedProxies)
	}
}

// TestProductionRequiresStrongEnvironmentConfiguration 验证生产环境的密钥、HTTPS 和数据库密码约束。
func TestProductionRequiresStrongEnvironmentConfiguration(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "short")
	if _, err := Load(); err == nil {
		t.Fatal("production accepted a short JWT secret")
	}

	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("WEB_BASE_URL", "http://tokenpulse.example.com")
	if _, err := Load(); err == nil {
		t.Fatal("production accepted an insecure public URL")
	}

	t.Setenv("WEB_BASE_URL", "https://tokenpulse.example.com")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://tokenpulse.example.com")
	t.Setenv("MYSQL_PASSWORD", "")
	if _, err := Load(); err == nil {
		t.Fatal("production accepted a missing database password")
	}
}
