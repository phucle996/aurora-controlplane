package iam_handler

import (
	"controlplane/internal/http/response"
	iam_domainsvc "controlplane/internal/iam/domain/service"
	iam_errorx "controlplane/internal/iam/errorx"
	iam_reqdto "controlplane/internal/iam/transport/http/request"
	"controlplane/pkg/logger"
	"errors"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc iam_domainsvc.AuthService
}

func NewAuthHandler(authSvc iam_domainsvc.AuthService) *AuthHandler {
	return &AuthHandler{
		authSvc: authSvc,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req iam_reqdto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.auth", "bind_error", "Failed to bind request payload: "+err.Error(), "")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	token, err := h.authSvc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		logger.HandlerError(c, "iam.auth", "login_failed", "Login attempt failed: "+err.Error(), "")

		if errors.Is(err, iam_errorx.ErrUserInactive) {
			response.RespondForbidden(c, "account is inactive")
			return
		}

		if errors.Is(err, iam_errorx.ErrInvalidCredentials) || errors.Is(err, iam_errorx.ErrUserNotFound) {
			response.RespondUnauthorized(c, "invalid email or password")
			return
		}

		response.RespondInternalError(c, "an unexpected error occurred during login")
		return
	}

	logger.HandlerInfo(c, "iam.auth", "login_success", "User logged in successfully")
	response.RespondSuccess(c, gin.H{"token": token}, "login successful")
}
