package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const ClaimsKey contextKey = "jwt_claims"

type JWTValidator struct {
	secret     []byte
	issuer     string
	allowedAlg map[string]bool
}

func NewJWTValidator(secret, issuer string, allowedAlgorithms []string) *JWTValidator {
	algs := make(map[string]bool)
	for _, a := range allowedAlgorithms {
		algs[a] = true
	}
	if len(algs) == 0 {
		algs["HS256"] = true
	}
	return &JWTValidator{
		secret:     []byte(secret),
		issuer:     issuer,
		allowedAlg: algs,
	}
}

func (j *JWTValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			alg, _ := token.Header["alg"].(string)
			if !j.allowedAlg[alg] {
				return nil, fmt.Errorf("disallowed algorithm: %s", alg)
			}
			return j.secret, nil
		},
			jwt.WithIssuer(j.issuer),
			jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}),
		)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		if !token.Valid {
			http.Error(w, `{"error":"token is not valid"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"error":"could not parse claims"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetClaims(r *http.Request) (jwt.MapClaims, bool) {
	claims, ok := r.Context().Value(ClaimsKey).(jwt.MapClaims)
	return claims, ok
}
