package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTService(secret string, ttl time.Duration) *JWTService {
	return &JWTService{secret: []byte(secret), ttl: ttl}
}

type claims struct {
	jwt.RegisteredClaims
}

// Issue mints a signed access token scoped to a single admin user id.
func (s *JWTService) Issue(adminUserID uuid.UUID) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   adminUserID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign jwt: %w", err)
	}
	return signed, expiresAt, nil
}

// Parse validates a bearer token and extracts the admin user id.
func (s *JWTService) Parse(raw string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(raw, &claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse jwt: %w", err)
	}
	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}
	adminUserID, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid subject: %w", err)
	}
	return adminUserID, nil
}
