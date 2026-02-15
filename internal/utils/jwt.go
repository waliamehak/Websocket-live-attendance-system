package utils

import (
	"errors"
	"fmt"
	"os"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string
	Role   string
}

var jwks *keyfunc.JWKS

func InitJWKS() error {
	domain := os.Getenv("AUTH0_DOMAIN")
	jwksURL := fmt.Sprintf("https://%s/.well-known/jwks.json", domain)

	var err error
	jwks, err = keyfunc.Get(jwksURL, keyfunc.Options{})
	return err
}

func ValidateToken(tokenString string) (*Claims, error) {
	if jwks == nil {
		return nil, errors.New("JWKS not initialized")
	}

	domain := os.Getenv("AUTH0_DOMAIN")
	audience := os.Getenv("AUTH0_AUDIENCE")
	namespace := os.Getenv("AUTH0_NAMESPACE")
	issuer := fmt.Sprintf("https://%s/", domain)

	token, err := jwt.Parse(tokenString, jwks.Keyfunc,
		jwt.WithAudience(audience),
		jwt.WithIssuer(issuer),
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	sub, _ := mapClaims["sub"].(string)
	role, _ := mapClaims[namespace+"/role"].(string)

	return &Claims{UserID: sub, Role: role}, nil
}
