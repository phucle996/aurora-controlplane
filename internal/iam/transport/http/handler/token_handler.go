package iam_handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
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
//  3. Set the new tokens as HttpOnly cookies and return 204 No Content.

// @Router /api/v1/auth/refresh [post]
// @Tags Token
// @Summary Refresh token
// @Description Refresh token
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
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
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	deviceIDCookie, err := c.Cookie("device_id")
	if err != nil {
		logger.HandlerWarn(c, "iam.token.refresh", err, "device id cookie not found")
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	if strings.TrimSpace(req.DeviceID) != strings.TrimSpace(deviceIDCookie) {
		logger.HandlerWarn(c, "iam.token.refresh", nil, "device id mismatch")
		response.RespondUnauthorized(c, "refresh token is invalid or expired")
		return
	}

	result, err := h.tokenSvc.Rotate(ctx, &iam_domainsvc.RotateRequest{
		RawRefreshToken: refreshTokenCookie,
		DeviceID:        deviceIDCookie,
		JTI:             req.JTI,
		IssuedAt:        req.IssuedAt,
		HTM:             req.HTM,
		HTU:             req.HTU,
		TokenHash:       req.TokenHash,
		Signature:       req.Signature,
	})
	if err != nil {
		logger.HandlerError(c, "iam.token.refresh", err)
		switch {
		case errors.Is(err, iam_errorx.ErrRefreshTokenInvalid),
			errors.Is(err, iam_errorx.ErrRefreshTokenMismatch),
			errors.Is(err, iam_errorx.ErrRefreshSignatureReplay),
			errors.Is(err, iam_errorx.ErrRefreshDeviceUnbound),
			errors.Is(err, iam_errorx.ErrRefreshSignatureInvalid),
			errors.Is(err, iam_errorx.ErrRefreshSignatureExpired):
			response.RespondUnauthorized(c, "refresh token is invalid or expired")
		default:
			response.RespondInternalError(c, "internal server error")
		}
		return
	}

	setSessionCookies(c, result.AccessToken, result.RefreshToken, result.DeviceID, result.AccessTokenExpiresAt, result.RefreshTokenExpiresAt)

	logger.HandlerInfo(c, "iam.token.refresh", "token rotated")
	c.AbortWithStatus(http.StatusNoContent)
}
