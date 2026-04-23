package iam_handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"controlplane/internal/iam/domain/entity"
	iam_domainsvc "controlplane/internal/iam/domain/service"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

type stubMfaHandlerService struct {
	verifyUserID   string
	verifyDeviceID string
}

func (s *stubMfaHandlerService) CheckAndChallenge(ctx context.Context, userID, deviceID string) (bool, string, []string, error) {
	return false, "", nil, nil
}

func (s *stubMfaHandlerService) Verify(ctx context.Context, challengeID, method, code string) (string, string, error) {
	if s.verifyUserID == "" {
		s.verifyUserID = "user-1"
	}
	if s.verifyDeviceID == "" {
		s.verifyDeviceID = "device-1"
	}
	return s.verifyUserID, s.verifyDeviceID, nil
}

func (s *stubMfaHandlerService) EnrollTOTP(ctx context.Context, userID, deviceName string) (string, string, error) {
	return "", "", nil
}

func (s *stubMfaHandlerService) ConfirmTOTP(ctx context.Context, userID, settingID, code string) error {
	return nil
}

func (s *stubMfaHandlerService) ListMethods(ctx context.Context, userID string) ([]*entity.MfaSetting, error) {
	return nil, nil
}

func (s *stubMfaHandlerService) EnableMethod(ctx context.Context, userID, settingID string) error {
	return nil
}

func (s *stubMfaHandlerService) DisableMethod(ctx context.Context, userID, settingID string) error {
	return nil
}

func (s *stubMfaHandlerService) DeleteMethod(ctx context.Context, userID, settingID string) error {
	return nil
}

func (s *stubMfaHandlerService) GenerateRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}

func TestMfaHandlerVerifySetsCookiesOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()
	h := NewMfaHandler(&stubMfaHandlerService{}, &stubTokenHandlerService{
		rotateResult: &iam_domainsvc.TokenResult{
			AccessToken:           "access-token",
			RefreshToken:          "refresh-token",
			DeviceID:              "device-1",
			AccessTokenExpiresAt:  time.Now().Add(time.Minute),
			RefreshTokenExpiresAt: time.Now().Add(2 * time.Minute),
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", bytes.NewReader([]byte(`{"challenge_id":"challenge-1","method":"totp","code":"123456"}`)))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.Verify(c)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for successful mfa verify, got %d", w.Code)
	}
	if len(bytes.TrimSpace(w.Body.Bytes())) != 0 {
		t.Fatalf("expected empty body for cookie-only mfa verify, got %s", w.Body.String())
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
