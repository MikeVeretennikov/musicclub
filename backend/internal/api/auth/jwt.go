package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"musicclubbot/backend/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWT configuration
const (
	AccessTokenExp   = 15 * time.Minute   // 15 minutes
	RefreshTokenExp  = 7 * 24 * time.Hour // 7 days
	RefreshTokenSize = 32                 // bytes for refresh token
)

type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Refresh tokens table structure
type RefreshToken struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Token     string    `db:"token"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

func GenerateAccessToken(ctx context.Context, userID uuid.UUID, username string) (string, error) {
	cfg := ctx.Value("cfg").(config.Config)
	expirationTime := time.Now().Add(AccessTokenExp)

	claims := &JWTClaims{
		UserID:   userID.String(),
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "musicclubbot",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(cfg.JwtSecretKey)
}

func GenerateRefreshToken() (string, error) {
	// Generate a secure random string for refresh token
	tokenBytes := make([]byte, RefreshTokenSize)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

func VerifyToken(ctx context.Context, tokenString string) (*JWTClaims, error) {
	cfg := ctx.Value("cfg").(config.Config)
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return cfg.JwtSecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
