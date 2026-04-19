package iam_handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"controlplane/internal/http/middleware"
	"controlplane/internal/http/response"
	"controlplane/internal/iam/domain/entity"
	iam_domainsvc "controlplane/internal/iam/domain/service"
	iam_errorx "controlplane/internal/iam/errorx"
	iam_reqdto "controlplane/internal/iam/transport/http/request"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

// DeviceHandler handles device management endpoints.
type DeviceHandler struct {
	deviceSvc iam_domainsvc.DeviceService
}

func NewDeviceHandler(deviceSvc iam_domainsvc.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceSvc: deviceSvc}
}

// ── Security ──────────────────────────────────────────────────────────────────

// IssueChallenge POST /devices/challenge
func (h *DeviceHandler) IssueChallenge(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	claims, ok := middleware.JWTClaims(c)
	if !ok {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	var req iam_reqdto.IssueChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.device.challenge", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	ch, err := h.deviceSvc.IssueChallenge(ctx, claims.Subject, strings.TrimSpace(req.DeviceID))
	if err != nil {
		logger.HandlerError(c, "iam.device.challenge", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.device.challenge", "challenge issued")
	response.RespondSuccess(c, gin.H{
		"challenge_id": ch.ChallengeID,
		"nonce":        ch.Nonce,
		"expires_at":   ch.ExpiresAt,
	}, "challenge issued")
}

// VerifyProof POST /devices/verify
func (h *DeviceHandler) VerifyProof(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req iam_reqdto.VerifyProofRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.device.verify", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	proof := proofFromReq(req)

	if err := h.deviceSvc.VerifyProof(ctx, proof); err != nil {
		logger.HandlerError(c, "iam.device.verify", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.device.verify", "proof verified")
	response.RespondSuccess(c, nil, "device proof verified")
}

// RotateKey POST /devices/rotate-key
func (h *DeviceHandler) RotateKey(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	claims, ok := middleware.JWTClaims(c)
	if !ok {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	var req iam_reqdto.RotateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.device.rotate-key", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	if err := h.deviceSvc.RotateKey(ctx, claims.Subject, req.DeviceID, req.NewPublicKey, req.NewAlgorithm); err != nil {
		logger.HandlerError(c, "iam.device.rotate-key", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.device.rotate-key", "device key rotated")
	response.RespondSuccess(c, nil, "device key rotated")
}

// Rebind POST /devices/rebind
func (h *DeviceHandler) Rebind(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	claims, ok := middleware.JWTClaims(c)
	if !ok {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	var req iam_reqdto.RebindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.device.rebind", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	// Verify proof before rebind
	proof := proofFromRebindReq(req)
	if err := h.deviceSvc.VerifyProof(ctx, proof); err != nil {
		logger.HandlerError(c, "iam.device.rebind", err)
		h.mapDeviceError(c, err)
		return
	}

	if err := h.deviceSvc.Rebind(ctx, claims.Subject, req.DeviceID, req.NewPublicKey, req.NewAlgorithm); err != nil {
		logger.HandlerError(c, "iam.device.rebind", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.device.rebind", "device rebound")
	response.RespondSuccess(c, nil, "device rebound successfully")
}

// RevokeDevice DELETE /devices/:device_id/revoke  (security group)
func (h *DeviceHandler) RevokeDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	claims, ok := middleware.JWTClaims(c)
	if !ok {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	deviceID := strings.TrimSpace(c.Param("device_id"))
	if deviceID == "" {
		response.RespondBadRequest(c, "device_id is required")
		return
	}

	if err := h.deviceSvc.Revoke(ctx, claims.Subject, deviceID); err != nil {
		logger.HandlerError(c, "iam.device.revoke", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.device.revoke", "device revoked")
	response.RespondSuccess(c, nil, "device revoked")
}

// Quarantine POST /devices/:device_id/quarantine  (security group)
func (h *DeviceHandler) Quarantine(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	deviceID := strings.TrimSpace(c.Param("device_id"))
	if deviceID == "" {
		response.RespondBadRequest(c, "device_id is required")
		return
	}

	if err := h.deviceSvc.Quarantine(ctx, deviceID); err != nil {
		logger.HandlerError(c, "iam.device.quarantine", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.device.quarantine", "device quarantined")
	response.RespondSuccess(c, nil, "device quarantined")
}

// ── User self-service ─────────────────────────────────────────────────────────

// ListMyDevices GET /api/v1/me/devices
func (h *DeviceHandler) ListMyDevices(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	claims, ok := middleware.JWTClaims(c)
	if !ok {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	devices, err := h.deviceSvc.ListByUserID(ctx, claims.Subject)
	if err != nil {
		logger.HandlerError(c, "iam.device.list", err)
		response.RespondInternalError(c, "failed to retrieve devices")
		return
	}

	response.RespondSuccess(c, devices, "")
}

// RenameDevice PATCH /api/v1/me/devices/:device_id/name
func (h *DeviceHandler) RenameDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	claims, ok := middleware.JWTClaims(c)
	if !ok {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	deviceID := strings.TrimSpace(c.Param("device_id"))
	if deviceID == "" {
		response.RespondBadRequest(c, "device_id is required")
		return
	}

	var req iam_reqdto.RenameDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.device.rename", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	if err := h.deviceSvc.Rename(ctx, claims.Subject, deviceID, req.Name); err != nil {
		logger.HandlerError(c, "iam.device.rename", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.device.rename", "device renamed")
	response.RespondSuccess(c, nil, "device renamed")
}

// RevokeOneDevice DELETE /api/v1/me/devices/:device_id
func (h *DeviceHandler) RevokeOneDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	claims, ok := middleware.JWTClaims(c)
	if !ok {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	deviceID := strings.TrimSpace(c.Param("device_id"))
	if deviceID == "" {
		response.RespondBadRequest(c, "device_id is required")
		return
	}

	if err := h.deviceSvc.RevokeOne(ctx, claims.Subject, deviceID); err != nil {
		logger.HandlerError(c, "iam.device.revoke-one", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.device.revoke-one", "device revoked")
	response.RespondSuccess(c, nil, "device revoked")
}

// RevokeOtherDevices DELETE /api/v1/me/devices/others
func (h *DeviceHandler) RevokeOtherDevices(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	claims, ok := middleware.JWTClaims(c)
	if !ok {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	// The "current" device is optionally passed as a query param so the caller
	// can keep itself alive.
	keepDeviceID := strings.TrimSpace(c.Query("keep_device_id"))
	if keepDeviceID == "" {
		response.RespondBadRequest(c, "keep_device_id query param is required")
		return
	}

	n, err := h.deviceSvc.RevokeOthers(ctx, claims.Subject, keepDeviceID)
	if err != nil {
		logger.HandlerError(c, "iam.device.revoke-others", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.device.revoke-others", "other devices revoked")
	response.RespondSuccess(c, gin.H{"revoked": n}, "other devices revoked")
}

// ── Admin / internal ──────────────────────────────────────────────────────────

// AdminGetDevice GET /admin/devices/:device_id
func (h *DeviceHandler) AdminGetDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	deviceID := strings.TrimSpace(c.Param("device_id"))
	if deviceID == "" {
		response.RespondBadRequest(c, "device_id is required")
		return
	}

	device, err := h.deviceSvc.AdminGetByID(ctx, deviceID)
	if err != nil {
		logger.HandlerError(c, "iam.admin.device.get", err)
		h.mapDeviceError(c, err)
		return
	}

	response.RespondSuccess(c, device, "")
}

// AdminForceRevoke DELETE /admin/devices/:device_id
func (h *DeviceHandler) AdminForceRevoke(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	deviceID := strings.TrimSpace(c.Param("device_id"))
	if deviceID == "" {
		response.RespondBadRequest(c, "device_id is required")
		return
	}

	if err := h.deviceSvc.AdminRevoke(ctx, deviceID); err != nil {
		logger.HandlerError(c, "iam.admin.device.revoke", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.admin.device.revoke", "device force-revoked")
	response.RespondSuccess(c, nil, "device force-revoked")
}

// AdminMarkSuspicious PATCH /admin/devices/:device_id/suspicious
func (h *DeviceHandler) AdminMarkSuspicious(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	deviceID := strings.TrimSpace(c.Param("device_id"))
	if deviceID == "" {
		response.RespondBadRequest(c, "device_id is required")
		return
	}

	var req iam_reqdto.AdminMarkSuspiciousRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.admin.device.suspicious", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	if err := h.deviceSvc.MarkSuspicious(ctx, deviceID, req.Suspicious); err != nil {
		logger.HandlerError(c, "iam.admin.device.suspicious", err)
		h.mapDeviceError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.admin.device.suspicious", "device suspicious flag updated")
	response.RespondSuccess(c, nil, "device updated")
}

// AdminCleanupStale DELETE /admin/devices/stale
func (h *DeviceHandler) AdminCleanupStale(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	var req iam_reqdto.CleanupStaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "iam.admin.device.cleanup", err, "invalid payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	before := time.Now().UTC().AddDate(0, 0, -req.InactiveDays)
	n, err := h.deviceSvc.CleanupStale(ctx, before)
	if err != nil {
		logger.HandlerError(c, "iam.admin.device.cleanup", err)
		response.RespondInternalError(c, "cleanup failed")
		return
	}

	logger.HandlerInfo(c, "iam.admin.device.cleanup", "stale devices cleaned")
	response.RespondSuccess(c, gin.H{"removed": n}, "stale devices cleaned")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *DeviceHandler) mapDeviceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, iam_errorx.ErrDeviceNotFound):
		response.RespondNotFound(c, "device not found")
	case errors.Is(err, iam_errorx.ErrDeviceForbidden):
		response.RespondForbidden(c, "access denied")
	case errors.Is(err, iam_errorx.ErrDeviceSuspicious):
		response.RespondForbidden(c, "device is flagged suspicious")
	case errors.Is(err, iam_errorx.ErrDeviceChallengeInvalid),
		errors.Is(err, iam_errorx.ErrDeviceChallengeNotFound):
		response.RespondBadRequest(c, "challenge invalid or expired")
	case errors.Is(err, iam_errorx.ErrDeviceProofInvalid):
		response.RespondBadRequest(c, "device proof invalid")
	case errors.Is(err, iam_errorx.ErrDeviceKeyRotateFailed):
		response.RespondInternalError(c, "key rotation failed")
	default:
		response.RespondInternalError(c, "an unexpected error occurred")
	}
}

func proofFromReq(req iam_reqdto.VerifyProofRequest) *entity.DeviceProof {
	return &entity.DeviceProof{
		ChallengeID: req.ChallengeID,
		DeviceID:    req.DeviceID,
		Signature:   req.Signature,
	}
}

func proofFromRebindReq(req iam_reqdto.RebindRequest) *entity.DeviceProof {
	return &entity.DeviceProof{
		ChallengeID: req.ChallengeID,
		DeviceID:    req.DeviceID,
		Signature:   req.Signature,
	}
}
