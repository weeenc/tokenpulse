// 本文件验证密码、JWT 和随机凭据工具的安全契约。
package security

import (
	"testing"
	"time"
)

// TestPasswordAndJWT 覆盖密码校验以及 JWT 类型隔离。
func TestPasswordAndJWT(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("password should match")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password should not match")
	}
	token, err := NewJWT(42, "access", "test-secret", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseJWT(token, "access", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 {
		t.Fatalf("unexpected user id: %d", claims.UserID)
	}
	if _, err := ParseJWT(token, "refresh", "test-secret"); err == nil {
		t.Fatal("access token must not be accepted as refresh")
	}
}

// TestRandomTokensAndCodes 验证随机令牌、摘要和用户码的外部格式。
func TestRandomTokensAndCodes(t *testing.T) {
	token, err := RandomToken("dt_", 32)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) < 40 || token[:3] != "dt_" {
		t.Fatalf("unexpected token shape")
	}
	if len(SHA256(token)) != 64 {
		t.Fatal("sha256 must be 64 hex characters")
	}
	code, err := UserCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 9 || code[4] != '-' {
		t.Fatalf("unexpected user code: %s", code)
	}
}
