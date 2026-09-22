package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"task-management/internal/config"
	"task-management/internal/constants"
	"task-management/internal/dtos"
	"task-management/internal/models"
)

// CustomClaims encapsulates application-specific payload fields in standard JWT claims.
type CustomClaims struct {
	UserID    uuid.UUID `json:"sub"`
	SessionID string    `json:"sid"`
	Email     string    `json:"email"`
	TeamID    uuid.UUID `json:"team_id"`
	TokenType string    `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// GenerateTokenPair signs and returns a new Access Token and Refresh Token pair for a user session.
func GenerateTokenPair(cfg config.Config, user *models.User, sessionID string) (*dtos.TokenResponse, error) {
	now := time.Now()
	accessExp := now.Add(cfg.JWTAccessExpiration)
	refreshExp := now.Add(cfg.JWTRefreshExpiration)

	// 1. Access Token
	accessClaims := CustomClaims{
		UserID:    user.ID,
		SessionID: sessionID,
		Email:     user.Email,
		TeamID:    user.TeamID,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    cfg.AppName,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := accessTokenObj.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// 2. Refresh Token
	refreshClaims := CustomClaims{
		UserID:    user.ID,
		SessionID: sessionID,
		Email:     user.Email,
		TeamID:    user.TeamID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    cfg.AppName,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := refreshTokenObj.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &dtos.TokenResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int64(cfg.JWTAccessExpiration.Seconds()),
		RefreshExpiresIn: int64(cfg.JWTRefreshExpiration.Seconds()),
	}, nil
}

// ValidateToken parses and verifies the signature and expiration of a JWT token string.
func ValidateToken(cfg config.Config, tokenStr string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, constants.ErrInvalidToken
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, constants.ErrInvalidToken
	}

	return claims, nil
}

