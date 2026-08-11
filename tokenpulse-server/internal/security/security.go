// Package security 封装密码、JWT、随机令牌和摘要等安全基础能力。
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims 是 TokenPulse JWT 的业务声明和标准注册声明集合。
type Claims struct {
	UserID               uint64 `json:"uid"` // UserID 标识令牌所属用户。
	Type                 string `json:"typ"` // Type 区分 access 等令牌用途，防止跨类型复用。
	jwt.RegisteredClaims        // RegisteredClaims 提供签发方、受众、签发和过期时间等标准声明。
}

// HashPassword 使用 bcrypt 默认成本计算不可逆密码哈希。
func HashPassword(password string) (string, error) {
	result, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(result), nil
}

// CheckPassword 使用恒定时间实现验证明文密码是否匹配 bcrypt 哈希。
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// NewJWT 使用 HS256 签发包含用户、用途、签发方、受众和有效期的 JWT。
func NewJWT(userID uint64, kind, secret string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID: userID,
		Type:   kind,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer: "tokenpulse", Subject: fmt.Sprintf("%d", userID), Audience: jwt.ClaimStrings{"tokenpulse-web"},
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseJWT 验证签名算法、签发方、受众、有效期及业务令牌类型。
func ParseJWT(value, kind, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(value, &Claims{}, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("tokenpulse"), jwt.WithAudience("tokenpulse-web"))
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || claims.Type != kind {
		return nil, fmt.Errorf("invalid token type")
	}
	return claims, nil
}

// RandomToken 从密码学安全随机源生成 URL 安全令牌，并添加可识别前缀。
func RandomToken(prefix string, bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(buffer), nil
}

// SHA256 返回输入字符串的 64 位小写十六进制 SHA-256 摘要。
func SHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// UserCode 生成易人工输入的八字符设备授权码，中间使用连字符分组。
// 字母表主动排除 I、L、O、0、1 等容易混淆的字符。
func UserCode() (string, error) {
	const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate user code: %w", err)
	}
	var b strings.Builder
	for index, value := range raw {
		if index == 4 {
			b.WriteByte('-')
		}
		b.WriteByte(alphabet[int(value)%len(alphabet)])
	}
	return b.String(), nil
}
