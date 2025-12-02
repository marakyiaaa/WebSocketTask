package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingToken = errors.New("missing bearer token")
	ErrInvalidToken = errors.New("invalid token")
)

type Claims struct {
	UserID string `json:"sub"`
	jwt.RegisteredClaims
}

type Validator struct {
	secret    []byte
	issuer    string
	audience  string
	validator *jwt.Validator
}

func NewValidator(secret, issuer, audience string, leeway time.Duration) *Validator {
	if leeway <= 0 {
		leeway = 0
	}

	return &Validator{
		secret:    []byte(secret),
		issuer:    issuer,
		audience:  audience,
		validator: jwt.NewValidator(jwt.WithLeeway(leeway)),
	}
}

func (v *Validator) Validate(ctx context.Context, rawToken string) (*Claims, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(rawToken) == "" {
		return nil, ErrMissingToken
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)

	claims := &Claims{}
	token, err := parser.ParseWithClaims(rawToken, claims, func(t *jwt.Token) (interface{}, error) {
		return v.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	if v.issuer != "" && claims.Issuer != v.issuer {
		return nil, ErrInvalidToken
	}

	if v.audience != "" && !contains(claims.Audience, v.audience) {
		return nil, ErrInvalidToken
	}

	if err := v.validator.Validate(claims); err != nil {
		return nil, fmt.Errorf("claims validation failed: %w", err)
	}

	if claims.UserID == "" {
		return nil, fmt.Errorf("claims missing subject")
	}

	return claims, nil
}

func contains(list jwt.ClaimStrings, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func ExtractBearerToken(header string) (string, error) {
	if header == "" {
		return "", ErrMissingToken
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrInvalidToken
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", ErrInvalidToken
	}

	return token, nil
}
