package iam_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"controlplane/internal/iam/domain/entity"
	iam_errorx "controlplane/internal/iam/errorx"
	iam_reqdto "controlplane/internal/iam/transport/http/request"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

type stubAuthHandlerService struct {
	loginCalled  bool
	loginErr     error
	loginResult  *entity.LoginResult
	adminLoginCalled bool
	adminLoginErr    error
	whoAmIResult *entity.WhoAmI
	whoAmIErr    error
}

func (s *stubAuthHandlerService) Login(ctx context.Context, username, password, deviceFingerprint, devicePublicKey, deviceKeyAlgorithm string) (*entity.LoginResult, error) {
	s.loginCalled = true
	if s.loginResult != nil {
		return s.loginResult, s.loginErr
	}
	return &entity.LoginResult{}, s.loginErr
}
func (s *stubAuthHandlerService) AdminAPIKeyLogin(ctx context.Context, apiKey string) error {
	s.adminLoginCalled = true
	return s.adminLoginErr
}
func (s *stubAuthHandlerService) Register(ctx context.Context, user *entity.User, profile *entity.UserProfile, rawPassword string) error {
	return nil
}
func (s *stubAuthHandlerService) WhoAmI(ctx context.Context, userID string) (*entity.WhoAmI, error) {
	if s.whoAmIErr != nil {
		return nil, s.whoAmIErr
	}
	if s.whoAmIResult != nil {
		return s.whoAmIResult, nil
	}
	return &entity.WhoAmI{}, nil
}
func (s *stubAuthHandlerService) Activate(ctx context.Context, token string) error       { return nil }
func (s *stubAuthHandlerService) ForgotPassword(ctx context.Context, email string) error { return nil }
func (s *stubAuthHandlerService) ResetPassword(ctx context.Context, token, newPassword string) error {
	return nil
}
func (s *stubAuthHandlerService) Logout(ctx context.Context, jti string, rawRefreshToken string) error {
	return nil
}

func TestAuthHandlerLoginRejectsMissingDeviceBindingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()
	svc := &stubAuthHandlerService{}
	h := NewAuthHandler(svc)

	body, err := json.Marshal(gin.H{
		"username": "user-1",
		"password": "password123",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.Login(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing binding fields, got %d", w.Code)
	}
	if svc.loginCalled {
		t.Fatalf("expected login service not to be called when binding fields are missing")
	}
}

func TestAuthHandlerLoginMapsDeviceBindingErrorsToBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()
	svc := &stubAuthHandlerService{loginErr: iam_errorx.ErrDeviceKeyInvalid}
	h := NewAuthHandler(svc)

	reqBody := iam_reqdto.LoginRequest{
		Username:           "user-1",
		Password:           "password123",
		DeviceFingerprint:  "install-abc",
		DevicePublicKey:    "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----",
		DeviceKeyAlgorithm: "ES256",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.Login(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for device binding error, got %d", w.Code)
	}
	if !svc.loginCalled {
		t.Fatalf("expected login service to be called for valid payload")
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("invalid request payload")) {
		t.Fatalf("expected generic bad request response, got %s", w.Body.String())
	}
}

func TestAuthHandlerLoginSetsCookiesOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()
	svc := &stubAuthHandlerService{
		loginResult: &entity.LoginResult{
			AccessToken:           "access-token",
			RefreshToken:          "refresh-token",
			DeviceID:              "device-1",
			AccessTokenExpiresAt:  time.Now().Add(time.Minute),
			RefreshTokenExpiresAt: time.Now().Add(2 * time.Minute),
		},
	}
	h := NewAuthHandler(svc)

	reqBody := iam_reqdto.LoginRequest{
		Username:           "user-1",
		Password:           "password123",
		DeviceFingerprint:  "install-abc",
		DevicePublicKey:    "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----",
		DeviceKeyAlgorithm: "ES256",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.Login(c)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for successful login, got %d", w.Code)
	}
	if !svc.loginCalled {
		t.Fatalf("expected login service to be called")
	}
	if len(bytes.TrimSpace(w.Body.Bytes())) != 0 {
		t.Fatalf("expected empty body for cookie-only login, got %s", w.Body.String())
	}
	resp := w.Result()
	defer resp.Body.Close()
	cookies := map[string]*http.Cookie{}
	for _, cookie := range resp.Cookies() {
		cookies[cookie.Name] = cookie
	}
	for _, name := range []string{"access_token", "refresh_token", "device_id", "refresh_token_hash"} {
		if _, ok := cookies[name]; !ok {
			t.Fatalf("expected %s cookie to be set", name)
		}
	}
	if !cookies["access_token"].HttpOnly || !cookies["refresh_token"].HttpOnly {
		t.Fatalf("expected auth cookies to be HttpOnly")
	}
	if cookies["device_id"].HttpOnly || cookies["refresh_token_hash"].HttpOnly {
		t.Fatalf("expected companion cookies to be readable")
	}
}

func TestAuthHandlerWhoAmIReturnsFlatSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()
	svc := &stubAuthHandlerService{
		whoAmIResult: &entity.WhoAmI{
			UserID:      "user-1",
			Username:    "user-1",
			Email:       "user@example.com",
			Phone:       "123456789",
			FullName:    "User One",
			Status:      "active",
			OnBoarding:  false,
			Level:       1,
			AuthType:    "password",
			Roles:       []string{"admin"},
			Permissions: []string{"iam:users:read"},
		},
	}
	h := NewAuthHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/whoami", nil)
	c.Request = req
	c.Set("user_id", "user-1")
	c.Set("device_id", "device-1")

	h.WhoAmI(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for whoami, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"user_id"`)) {
		t.Fatalf("expected user_id in whoami response, got %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"full_name"`)) {
		t.Fatalf("expected full_name in whoami response, got %s", w.Body.String())
	}
}

func TestAuthHandlerAdminLoginSetsAPITokenCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()
	svc := &stubAuthHandlerService{}
	h := NewAuthHandler(svc)

	body, err := json.Marshal(gin.H{
		"api_key": "admin-api-key-1",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/admin/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.AdminLogin(c)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for successful admin login, got %d", w.Code)
	}
	if !svc.adminLoginCalled {
		t.Fatalf("expected admin login service to be called")
	}

	resp := w.Result()
	defer resp.Body.Close()
	var apiTokenCookie *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "apitoken" {
			apiTokenCookie = cookie
			break
		}
	}
	if apiTokenCookie == nil {
		t.Fatalf("expected apitoken cookie to be set")
	}
	if !apiTokenCookie.HttpOnly {
		t.Fatalf("expected apitoken cookie to be HttpOnly")
	}
}

func TestAuthHandlerAdminLoginInvalidKeyReturnsUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()
	svc := &stubAuthHandlerService{adminLoginErr: iam_errorx.ErrAdminAPIKeyInvalid}
	h := NewAuthHandler(svc)

	body, err := json.Marshal(gin.H{
		"api_key": "wrong-key",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/admin/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.AdminLogin(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid admin key, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("unauthorized")) {
		t.Fatalf("expected generic unauthorized response, got %s", w.Body.String())
	}
}

func TestAuthHandlerLogoutMissingRefreshCookieReturnsGenericUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()
	svc := &stubAuthHandlerService{}
	h := NewAuthHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	c.Request = req
	c.Set("jti", "jti-1")

	h.Logout(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing refresh cookie, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("unauthorized")) {
		t.Fatalf("expected generic unauthorized response, got %s", w.Body.String())
	}
}
