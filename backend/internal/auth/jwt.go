package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type Claims struct {
	jwt.RegisteredClaims
	CompanyID *uuid.UUID  `json:"company_id,omitempty"`
	Role      domain.Role `json:"role"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type JWTManager struct {
	secret []byte
}

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{secret: []byte(secret)}
}

func (m *JWTManager) Issue(user *domain.User) (*TokenPair, string, error) {
	now := time.Now()
	accessExp := now.Add(15 * time.Minute)

	accessClaims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
		CompanyID: user.CompanyID,
		Role:      user.Role,
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(m.secret)
	if err != nil {
		return nil, "", fmt.Errorf("sign access token: %w", err)
	}

	refreshRaw := make([]byte, 32)
	if _, err := rand.Read(refreshRaw); err != nil {
		return nil, "", fmt.Errorf("generate refresh token: %w", err)
	}
	refreshStr := hex.EncodeToString(refreshRaw)
	refreshHash := HashToken(refreshStr)

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresAt:    accessExp.Unix(),
	}, refreshHash, nil
}

func (m *JWTManager) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
