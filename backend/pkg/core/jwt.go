package core

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

func CreateToken(secret string, userID uuid.UUID, sessionID uuid.UUID, role string, expiration time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		SessionID: sessionID.String(),
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			ID:        uuid.NewString(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ParseToken(secret string, tokenString string) (Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return Claims{}, ErrUnauthorized
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.Subject == "" || claims.SessionID == "" || claims.Role == "" {
		return Claims{}, ErrUnauthorized
	}
	return *claims, nil
}
