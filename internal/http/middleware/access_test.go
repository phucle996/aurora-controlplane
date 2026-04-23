package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"controlplane/internal/security"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

type stubSecretProvider struct {
	candidates []security.SecretVersion
}

func (s *stubSecretProvider) GetActive(family string) (security.SecretVersion, error) {
	if len(s.candidates) == 0 {
		return security.SecretVersion{}, security.ErrSecretUnavailable
	}
	return s.candidates[0], nil
}

func (s *stubSecretProvider) GetCandidates(family string) ([]security.SecretVersion, error) {
	return s.candidates, nil
}

func TestAccessMiddlewareReadsAccessTokenCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()

	secret := "access-secret"
	Init(&stubSecretProvider{
		candidates: []security.SecretVersion{
			{Family: security.SecretFamilyAccess, Version: 1, Value: secret},
		},
	}, nil)
	t.Cleanup(func() {
		Init(nil, nil)
	})

	token, err := security.Sign(security.Claims{
		Subject:   "user-1",
		Role:      "user",
		Level:     4,
		Status:    "active",
		IssuedAt:  time.Now().UTC().Unix(),
		ExpiresAt: time.Now().UTC().Add(time.Hour).Unix(),
	}, secret)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/me/mfa", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	router := gin.New()
	router.Use(Access())
	router.GET("/api/v1/me/mfa", func(ctx *gin.Context) {
		if got := GetUserID(ctx); got != "user-1" {
			t.Fatalf("expected user id from cookie token, got %q", got)
		}
		ctx.Status(http.StatusNoContent)
	})

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected middleware to allow cookie token, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %s", w.Body.String())
	}
}
