package iam_handler

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"controlplane/internal/http/middleware"
	"controlplane/internal/http/response"
	"controlplane/internal/iam/domain/entity"
	iam_domainsvc "controlplane/internal/iam/domain/service"
	iam_errorx "controlplane/internal/iam/errorx"
	iam_reqdto "controlplane/internal/iam/transport/http/request"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

var (
	registerUsernamePattern = regexp.MustCompile(`^[a-z0-9._-]+$`)
)

type AuthHandler struct {
	authSvc iam_domainsvc.AuthService
}

func NewAuthHandler(authSvc iam_domainsvc.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req iam_reqdto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.auth.register", err, "Failed to bind register payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	fullName := strings.TrimSpace(req.FullName)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	username := strings.ToLower(strings.TrimSpace(req.Username))

	var phone string
	if req.PhoneNumber != nil {
		phone = strings.TrimSpace(*req.PhoneNumber)
	}

	if fullName == "" || utf8.RuneCountInString(fullName) < 2 || utf8.RuneCountInString(fullName) > 120 {
		logger.HandlerWarn(c, "iam.auth.register", nil, "invalid full name")
		response.RespondBadRequest(c, "invalid full name")
		return
	}
	if email == "" {
		logger.HandlerWarn(c, "iam.auth.register", nil, "invalid email")
		response.RespondBadRequest(c, "invalid email")
		return
	}
	if username == "" || len(username) < 3 || len(username) > 32 {
		logger.HandlerWarn(c, "iam.auth.register", nil, "invalid username")
		response.RespondBadRequest(c, "invalid username")
		return
	}
	if !registerUsernamePattern.MatchString(username) {
		logger.HandlerWarn(c, "iam.auth.register", nil, "invalid username")
		response.RespondBadRequest(c, "invalid username")
		return
	}

	if req.Password != req.RePassword {
		logger.HandlerWarn(c, "iam.auth.register", nil, "password confirmation does not match")
		response.RespondBadRequest(c, "password confirmation does not match")
		return
	}
	if len(req.Password) < 8 {
		logger.HandlerWarn(c, "iam.auth.register", nil, "weak password")
		response.RespondBadRequest(c, "weak password")
		return
	}

	user := &entity.User{
		Username:      username,
		Email:         email,
		Phone:         phone,
		PasswordHash:  "",
		SecurityLevel: 0,
		Status:        "pending",
		StatusReason:  "pending_email_verification",
		Role:          "",
	}
	profile := &entity.UserProfile{
		Fullname: fullName,
		Timezone: "UTC",
	}

	if err := h.authSvc.Register(ctx, user, profile, req.Password); err != nil {
		logger.HandlerError(c, "iam.auth.register", err)
		switch {
		case errors.Is(err, iam_errorx.ErrUsernameAlreadyExists),
			errors.Is(err, iam_errorx.ErrEmailAlreadyExists),
			errors.Is(err, iam_errorx.ErrPhoneAlreadyExists):
			response.RespondConflict(c, "account already exists")
		default:
			response.RespondInternalError(c, "an unexpected error occurred during registration")
		}
		return
	}

	logger.HandlerInfo(c, "iam.auth.register", "Account registered successfully")
	response.RespondCreated(c, nil, "Account registered successfully. Please verify your email.")
}

func (h *AuthHandler) Activate(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		logger.HandlerWarn(c, "iam.auth.activate", nil, "Missing activation token")
		response.RespondBadRequest(c, "invalid activation token")
		return
	}

	if err := h.authSvc.Activate(ctx, token); err != nil {
		logger.HandlerError(c, "iam.auth.activate", err)
		switch {
		case errors.Is(err, iam_errorx.ErrActivationTokenInvalid):
			response.RespondBadRequest(c, "invalid activation token")
		case errors.Is(err, iam_errorx.ErrActivationTokenExpired):
			response.RespondBadRequest(c, "activation token expired")
		case errors.Is(err, iam_errorx.ErrUserNotFound):
			response.RespondBadRequest(c, "invalid activation token")
		default:
			response.RespondInternalError(c, "an unexpected error occurred during activation")
		}
		return
	}

	logger.HandlerInfo(c, "iam.auth.activate", "Account activated successfully")
	response.RespondSuccess(c, nil, "account activated successfully")
}

func (h *AuthHandler) Login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req iam_reqdto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.auth", err, "Failed to bind request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	username := strings.ToLower(strings.TrimSpace(req.Username))
	password := strings.TrimSpace(req.Password)

	result, err := h.authSvc.Login(ctx, username, password)
	if err != nil {
		logger.HandlerError(c, "iam.auth", err)

		if errors.Is(err, iam_errorx.ErrUserInactive) {
			response.RespondForbidden(c, "account is inactive")
			return
		}

		if errors.Is(err, iam_errorx.ErrInvalidCredentials) || errors.Is(err, iam_errorx.ErrUserNotFound) {
			response.RespondUnauthorized(c, "invalid username or password")
			return
		}

		response.RespondInternalError(c, "an unexpected error occurred during login")
		return
	}

	if result != nil && result.Pending {
		logger.HandlerInfo(c, "iam.auth", "Pending account activation email resent")
		response.RespondAccepted(c, nil, "account is pending activation, verification email resent")
		return
	}

	// MFA gate — client must complete MFA before receiving tokens.
	if result != nil && result.MFARequired {
		logger.HandlerInfo(c, "iam.auth", "MFA required — challenge issued")
		response.RespondAccepted(c, gin.H{
			"mfa_required":      true,
			"challenge_id":      result.MFAChallengeID,
			"available_methods": result.MFAAvailableMethods,
		}, "mfa verification required")
		return
	}

	if result == nil {
		logger.HandlerError(c, "iam.auth", err)
		response.RespondInternalError(c, "an unexpected error occurred during login")
		return
	}

	secureCookie := c.Request != nil && (c.Request.TLS != nil || strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https"))
	accessMaxAge := int(time.Until(result.AccessTokenExpiresAt).Seconds())
	refreshMaxAge := int(time.Until(result.RefreshTokenExpiresAt).Seconds())

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    result.AccessToken,
		Path:     "/",
		MaxAge:   accessMaxAge,
		Expires:  result.AccessTokenExpiresAt,
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    result.RefreshToken,
		Path:     "/",
		MaxAge:   refreshMaxAge,
		Expires:  result.RefreshTokenExpiresAt,
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	logger.HandlerInfo(c, "iam.auth", "User logged in successfully")
	response.RespondSuccess(c, gin.H{
		"access_token":             result.AccessToken,
		"refresh_token":            result.RefreshToken,
		"device_id":                result.DeviceID,
		"access_token_expires_at":  result.AccessTokenExpiresAt.Unix(),
		"refresh_token_expires_at": result.RefreshTokenExpiresAt.Unix(),
	}, "login successful")
}

// ForgotPassword POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req iam_reqdto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.auth.forgot-password", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	// Always respond 202 — prevent enumeration of valid email addresses.
	if err := h.authSvc.ForgotPassword(ctx, req.Email); err != nil {
		logger.HandlerError(c, "iam.auth.forgot-password", err)
		// Do NOT expose to client.
	}

	logger.HandlerInfo(c, "iam.auth.forgot-password", "forgot-password requested")
	response.RespondAccepted(c, nil, "if the email is registered, a reset link has been sent")
}

// ResetPassword POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req iam_reqdto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.auth.reset-password", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	if req.NewPassword != req.RePassword {
		response.RespondBadRequest(c, "passwords do not match")
		return
	}

	token := c.Query("token")
	if token == "" {
		logger.HandlerWarn(c, "iam.auth.reset-password", nil, "invalid token")
		response.RespondBadRequest(c, "invalid token")
		return
	}

	if err := h.authSvc.ResetPassword(ctx, token, req.NewPassword); err != nil {
		logger.HandlerError(c, "iam.auth.reset-password", err)
		h.mapResetError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.auth.reset-password", "password reset successful")
	response.RespondSuccess(c, nil, "password reset successful")
}

func (h *AuthHandler) mapResetError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, iam_errorx.ErrResetTokenInvalid):
		response.RespondBadRequest(c, "reset token is invalid")
	case errors.Is(err, iam_errorx.ErrResetTokenExpired):
		response.RespondBadRequest(c, "reset token has expired")
	case errors.Is(err, iam_errorx.ErrWeakPassword):
		response.RespondBadRequest(c, "password does not meet requirements")
	default:
		response.RespondInternalError(c, "password reset failed")
	}
}

// Logout POST /api/v1/auth/logout
//
// Extracts access token JTI to blacklist it, and revokes the provided refresh token.
// Finally, clears the client cookies.
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 1. Get JTI from the gin context (injected by Access middleware)
	jti := c.GetString(middleware.CtxKeyJTI)

	// 2. Get refresh token from cookie
	refreshTokenCookie, err := c.Cookie("refresh_token")
	if err != nil {
		logger.HandlerWarn(c, "iam.auth.logout", err, "refresh token cookie not found")
		response.RespondUnauthorized(c, "refresh token cookie not found")
		return
	}
	if refreshTokenCookie == "" {
		logger.HandlerWarn(c, "iam.auth.logout", nil, "invalid token")
		response.RespondBadRequest(c, "invalid token")
		return
	}

	// 3. Blacklist access token and revoke refresh token
	if err := h.authSvc.Logout(ctx, jti, refreshTokenCookie); err != nil {
		logger.HandlerError(c, "iam.auth.logout", err)
		// Proceed anyway to clear cookies
	}

	// 4. Clear cookies on client
	secureCookie := c.Request.TLS != nil
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	logger.HandlerInfo(c, "iam.auth.logout", "user logged out successfully")
	response.RespondSuccess(c, nil, "logged out successfully")
}
