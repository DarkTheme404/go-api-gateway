package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func makeToken(secret string, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

func TestJWTValidator_MissingHeader(t *testing.T) {
	v := NewJWTValidator("secret", "test-issuer", nil)
	handler := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestJWTValidator_BadFormat(t *testing.T) {
	v := NewJWTValidator("secret", "test-issuer", nil)
	handler := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestJWTValidator_ValidToken(t *testing.T) {
	secret := "test-secret"
	v := NewJWTValidator(secret, "test-issuer", nil)
	handler := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaims(r)
		if !ok {
			t.Fatal("claims not found in context")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(claims["sub"].(string)))
	}))

	token := makeToken(secret, jwt.MapClaims{
		"iss": "test-issuer",
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "user-123" {
		t.Fatalf("expected user-123, got %s", rec.Body.String())
	}
}

func TestJWTValidator_InvalidToken(t *testing.T) {
	v := NewJWTValidator("secret", "test-issuer", nil)
	handler := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token := makeToken("wrong-secret", jwt.MapClaims{
		"iss": "test-issuer",
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
