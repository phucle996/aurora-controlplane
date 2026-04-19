package iam_handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"controlplane/internal/http/response"
	iam_domainsvc "controlplane/internal/iam/domain/service"
	iam_errorx "controlplane/internal/iam/errorx"
	iam_reqdto "controlplane/internal/iam/transport/http/request"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

// TokenHandler handles refresh-token issuance and rotation.
type TokenHandler struct {
	tokenSvc iam_domainsvc.TokenService
}

func NewTokenHandler(tokenSvc iam_domainsvc.TokenService) *TokenHandler {
	return &TokenHandler{tokenSvc: tokenSvc}
}

// Refresh POST /api/v1/auth/refresh
//
// Flow:
//  1. Bind and validate the signed request body.
//  2. Delegate to TokenService.Rotate which:
//     a. Verifies the device signature against the stored public key.
//     b. Revokes the presented refresh token.
//     c. Issues a new access token + refresh token.
//  3. Set the new tokens as HttpOnly cookies AND return them in the JSON body.
func (h *TokenHandler) Refresh(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req iam_reqdto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.token.refresh", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	refreshTokenCookie, err := c.Cookie("refresh_token")
	if err != nil {
		logger.HandlerWarn(c, "iam.token.refresh", err, "refresh token cookie not found")
		response.RespondUnauthorized(c, "refresh token cookie not found")
		return
	}

	deviceIDCookie, err := c.Cookie("device_id")
	if err != nil {
		logger.HandlerWarn(c, "iam.token.refresh", err, "device id cookie not found")
		response.RespondUnauthorized(c, "device id cookie not found")
		return
	}

	result, err := h.tokenSvc.Rotate(ctx, &iam_domainsvc.RotateRequest{
		RawRefreshToken: refreshTokenCookie,
		DeviceID:        deviceIDCookie,
		Nonce:           req.Nonce,
		TimestampUnix:   req.TimestampUnix,
		Signature:       req.Signature,
	})
	if err != nil {
		logger.HandlerError(c, "iam.token.refresh", err)
		switch {
		case errors.Is(err, iam_errorx.ErrRefreshTokenInvalid),
			errors.Is(err, iam_errorx.ErrRefreshTokenMismatch):
			response.RespondUnauthorized(c, "refresh token is invalid or expired")
		case errors.Is(err, iam_errorx.ErrRefreshSignatureInvalid):
			response.RespondUnauthorized(c, "request signature is invalid")
		case errors.Is(err, iam_errorx.ErrRefreshSignatureExpired):
			response.RespondUnauthorized(c, "signed request has expired — resend with a fresh timestamp")
		default:
			response.RespondInternalError(c, "token refresh failed")
		}
		return
	}

	// Set HttpOnly cookies so browser clients benefit automatically.
	secureCookie := c.Request.TLS != nil
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

	logger.HandlerInfo(c, "iam.token.refresh", "token rotated")
	response.RespondSuccess(c, gin.H{
		"access_token":             result.AccessToken,
		"refresh_token":            result.RefreshToken,
		"device_id":                result.DeviceID,
		"access_token_expires_at":  result.AccessTokenExpiresAt.Unix(),
		"refresh_token_expires_at": result.RefreshTokenExpiresAt.Unix(),
	}, "token refreshed")
}
