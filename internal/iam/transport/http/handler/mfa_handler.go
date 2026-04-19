package iam_handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"controlplane/internal/http/middleware"
	"controlplane/internal/http/response"
	iam_domainsvc "controlplane/internal/iam/domain/service"
	iam_errorx "controlplane/internal/iam/errorx"
	iam_reqdto "controlplane/internal/iam/transport/http/request"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

// MfaHandler handles all MFA-related HTTP endpoints.
type MfaHandler struct {
	mfaSvc   iam_domainsvc.MfaService
	tokenSvc iam_domainsvc.TokenService
}

func NewMfaHandler(
	mfaSvc iam_domainsvc.MfaService,
	tokenSvc iam_domainsvc.TokenService,
) *MfaHandler {
	return &MfaHandler{
		mfaSvc:   mfaSvc,
		tokenSvc: tokenSvc,
	}
}

// ── Challenge flow ────────────────────────────────────────────────────────────

// Verify POST /api/v1/auth/mfa/verify  (public — no access token needed)
//
// Client receives challengeID from LoginResult.MFAChallengeID.
// On success, new token pair is issued and returned.
func (h *MfaHandler) Verify(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	_ = ctx

	var req iam_reqdto.MfaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.mfa.verify", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	// Delegate full verification to MfaService.
	userID, deviceID, err := h.mfaSvc.Verify(ctx, req.ChallengeID, req.Method, req.Code)
	if err != nil {
		logger.HandlerError(c, "iam.mfa.verify", err)
		h.mapMfaError(c, err)
		return
	}

	// MFA passed — load user + device, then issue token pair via TokenService.
	// We pass a minimal entity.User with just the ID; TokenService.IssueAfterLogin
	// reconstructs full claims from its own fetch inside issueAccessToken.
	tokenResult, err := h.tokenSvc.IssueForMFA(ctx, userID, deviceID)
	if err != nil {
		logger.HandlerError(c, "iam.mfa.verify", err)
		response.RespondInternalError(c, "token issuance failed")
		return
	}

	secureCookie := c.Request.TLS != nil
	accessMaxAge := int(time.Until(tokenResult.AccessTokenExpiresAt).Seconds())
	refreshMaxAge := int(time.Until(tokenResult.RefreshTokenExpiresAt).Seconds())

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    tokenResult.AccessToken,
		Path:     "/",
		MaxAge:   accessMaxAge,
		Expires:  tokenResult.AccessTokenExpiresAt,
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokenResult.RefreshToken,
		Path:     "/",
		MaxAge:   refreshMaxAge,
		Expires:  tokenResult.RefreshTokenExpiresAt,
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	logger.HandlerInfo(c, "iam.mfa.verify", "mfa verified — tokens issued")
	response.RespondSuccess(c, gin.H{
		"access_token":             tokenResult.AccessToken,
		"refresh_token":            tokenResult.RefreshToken,
		"device_id":                tokenResult.DeviceID,
		"access_token_expires_at":  tokenResult.AccessTokenExpiresAt.Unix(),
		"refresh_token_expires_at": tokenResult.RefreshTokenExpiresAt.Unix(),
	}, "authentication successful")
}

// ── Self-service (requires access token) ─────────────────────────────────────

// ListMethods GET /api/v1/me/mfa
func (h *MfaHandler) ListMethods(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	_ = ctx

	userID := c.GetString(middleware.CtxKeyUserID)
	_ = userID
	methods, err := h.mfaSvc.ListMethods(ctx, userID)
	if err != nil {
		logger.HandlerError(c, "iam.mfa.list-methods", err)
		response.RespondInternalError(c, "failed to list mfa methods")
		return
	}

	logger.HandlerInfo(c, "iam.mfa.list-methods", "mfa methods listed")
	response.RespondSuccess(c, methods, "ok")
}

// EnrollTOTP POST /api/v1/me/mfa/totp/enroll
func (h *MfaHandler) EnrollTOTP(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	_ = ctx

	var req iam_reqdto.MfaEnrollTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	userID := c.GetString(middleware.CtxKeyUserID)
	_ = userID
	settingID, provisioningURI, err := h.mfaSvc.EnrollTOTP(ctx, userID, req.DeviceName)
	if err != nil {
		logger.HandlerError(c, "iam.mfa.enroll-totp", err)
		response.RespondInternalError(c, "totp enrollment failed")
		return
	}

	logger.HandlerInfo(c, "iam.mfa.enroll-totp", "totp enrolled")
	response.RespondSuccess(c, gin.H{
		"setting_id":       settingID,
		"provisioning_uri": provisioningURI,
	}, "scan the QR code and confirm with a valid code")
}

// ConfirmTOTP POST /api/v1/me/mfa/totp/confirm
func (h *MfaHandler) ConfirmTOTP(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	_ = ctx

	var req iam_reqdto.MfaConfirmTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	userID := c.GetString(middleware.CtxKeyUserID)
	_ = userID
	if err := h.mfaSvc.ConfirmTOTP(ctx, userID, req.SettingID, req.Code); err != nil {
		logger.HandlerError(c, "iam.mfa.confirm-totp", err)
		h.mapMfaError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.mfa.confirm-totp", "totp confirmed and enabled")
	response.RespondSuccess(c, nil, "totp enabled")
}

// EnableMethod PATCH /api/v1/me/mfa/:setting_id/enable
func (h *MfaHandler) EnableMethod(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	_ = ctx

	userID := c.GetString(middleware.CtxKeyUserID)
	_ = userID
	settingID := c.Param("setting_id")
	if err := h.mfaSvc.EnableMethod(ctx, userID, settingID); err != nil {
		logger.HandlerError(c, "iam.mfa.enable", err)
		h.mapMfaError(c, err)
		return
	}
	logger.HandlerInfo(c, "iam.mfa.enable", "mfa method enabled")
	response.RespondSuccess(c, nil, "method enabled")
}

// DisableMethod PATCH /api/v1/me/mfa/:setting_id/disable
func (h *MfaHandler) DisableMethod(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	_ = ctx

	userID := c.GetString(middleware.CtxKeyUserID)
	_ = userID
	settingID := c.Param("setting_id")
	if err := h.mfaSvc.DisableMethod(ctx, userID, settingID); err != nil {
		logger.HandlerError(c, "iam.mfa.disable", err)
		h.mapMfaError(c, err)
		return
	}
	logger.HandlerInfo(c, "iam.mfa.disable", "mfa method disabled")
	response.RespondSuccess(c, nil, "method disabled")
}

// DeleteMethod DELETE /api/v1/me/mfa/:setting_id
func (h *MfaHandler) DeleteMethod(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	_ = ctx

	userID := c.GetString(middleware.CtxKeyUserID)
	_ = userID
	settingID := c.Param("setting_id")
	if err := h.mfaSvc.DeleteMethod(ctx, userID, settingID); err != nil {
		logger.HandlerError(c, "iam.mfa.delete", err)
		h.mapMfaError(c, err)
		return
	}
	logger.HandlerInfo(c, "iam.mfa.delete", "mfa method removed")
	response.RespondSuccess(c, nil, "method removed")
}

// GenerateRecoveryCodes POST /api/v1/me/mfa/recovery-codes
func (h *MfaHandler) GenerateRecoveryCodes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	_ = ctx

	userID := c.GetString(middleware.CtxKeyUserID)
	_ = userID
	codes, err := h.mfaSvc.GenerateRecoveryCodes(ctx, userID)
	if err != nil {
		logger.HandlerError(c, "iam.mfa.recovery-codes", err)
		response.RespondInternalError(c, "recovery code generation failed")
		return
	}

	logger.HandlerInfo(c, "iam.mfa.recovery-codes", "recovery codes generated")
	// Codes returned once only — client must save them.
	c.JSON(http.StatusOK, gin.H{
		"recovery_codes": codes,
		"warning":        "these codes are shown only once — save them now",
	})
}

// ── error mapping ─────────────────────────────────────────────────────────────

func (h *MfaHandler) mapMfaError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, iam_errorx.ErrMfaChallengeNotFound),
		errors.Is(err, iam_errorx.ErrMfaChallengeInvalid):
		response.RespondUnauthorized(c, "mfa challenge is invalid or has expired")
	case errors.Is(err, iam_errorx.ErrMfaCodeInvalid):
		response.RespondUnauthorized(c, "mfa code is incorrect")
	case errors.Is(err, iam_errorx.ErrMfaCodeExpired):
		response.RespondUnauthorized(c, "mfa code has expired — request a new one")
	case errors.Is(err, iam_errorx.ErrMfaMethodNotAllowed):
		response.RespondBadRequest(c, "the selected mfa method is not available for this challenge")
	case errors.Is(err, iam_errorx.ErrMfaSettingNotFound):
		response.RespondNotFound(c, "mfa setting not found")
	default:
		response.RespondInternalError(c, "mfa operation failed")
	}
}
