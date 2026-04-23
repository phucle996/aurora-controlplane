package iam_handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"controlplane/internal/http/response"
	"controlplane/internal/iam/domain/entity"
	iam_domainsvc "controlplane/internal/iam/domain/service"
	iam_errorx "controlplane/internal/iam/errorx"
	iam_reqdto "controlplane/internal/iam/transport/http/request"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RbacHandler provides admin endpoints for roles and permissions.
// All routes must be protected by Access + RequireLevel middleware.
type RbacHandler struct {
	svc iam_domainsvc.RbacService
}

func NewRbacHandler(svc iam_domainsvc.RbacService) *RbacHandler {
	return &RbacHandler{svc: svc}
}

// ── Roles ─────────────────────────────────────────────────────────────────────

// @Router /api/v1/admin/rbac/roles [get]
// @Tags RBAC
// @Summary List roles
// @Description List roles
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
func (h *RbacHandler) ListRoles(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	roles, err := h.svc.ListRoles(ctx)
	if err != nil {
		logger.HandlerError(c, "iam.rbac.list-roles", err)
		response.RespondInternalError(c, "failed to list roles")
		return
	}
	logger.HandlerInfo(c, "iam.rbac.list-roles", "roles listed")
	response.RespondSuccess(c, roles, "ok")
}

// @Router /api/v1/admin/rbac/roles/:id [get]
// @Tags RBAC
// @Summary Get role
// @Description Get role
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
func (h *RbacHandler) GetRole(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rp, err := h.svc.GetRole(ctx, c.Param("id"))
	if err != nil {
		logger.HandlerError(c, "iam.rbac.get-role", err)
		h.mapError(c, err)
		return
	}
	logger.HandlerInfo(c, "iam.rbac.get-role", "role fetched")
	response.RespondSuccess(c, rp, "ok")
}

// @Router /api/v1/admin/rbac/roles [post]
// @Tags RBAC
// @Summary Create role
// @Description Create role
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
func (h *RbacHandler) CreateRole(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req iam_reqdto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	role := &entity.Role{
		Name:        req.Name,
		Level:       req.Level,
		Description: req.Description,
	}
	if err := h.svc.CreateRole(ctx, role); err != nil {
		logger.HandlerError(c, "iam.rbac.create-role", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "iam.rbac.create-role", "role created")
	c.JSON(http.StatusCreated, gin.H{"role": role, "message": "role created"})
}

// @Router /api/v1/admin/rbac/roles/:id [patch]
// @Tags RBAC
// @Summary Update role
// @Description Update role
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
func (h *RbacHandler) UpdateRole(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req iam_reqdto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	role := &entity.Role{
		ID:          c.Param("id"),
		Name:        req.Name,
		Level:       req.Level,
		Description: req.Description,
	}
	if err := h.svc.UpdateRole(ctx, role); err != nil {
		logger.HandlerError(c, "iam.rbac.update-role", err)
		h.mapError(c, err)
		return
	}
	logger.HandlerInfo(c, "iam.rbac.update-role", "role updated — cache invalidated")
	response.RespondSuccess(c, nil, "role updated — cache invalidated")
}

// @Router /api/v1/admin/rbac/roles/:id [delete]
// @Tags RBAC
// @Summary Delete role
// @Description Delete role
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
func (h *RbacHandler) DeleteRole(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.svc.DeleteRole(ctx, c.Param("id")); err != nil {
		logger.HandlerError(c, "iam.rbac.delete-role", err)
		h.mapError(c, err)
		return
	}
	logger.HandlerInfo(c, "iam.rbac.delete-role", "role deleted")
	response.RespondSuccess(c, nil, "role deleted")
}

// ── Permissions ───────────────────────────────────────────────────────────────

// @Router /api/v1/admin/rbac/permissions [get]
// @Tags RBAC
// @Summary List permissions
// @Description List permissions
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
func (h *RbacHandler) ListPermissions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	perms, err := h.svc.ListPermissions(ctx)
	if err != nil {
		logger.HandlerError(c, "iam.rbac.list-permissions", err)
		response.RespondInternalError(c, "failed to list permissions")
		return
	}
	logger.HandlerInfo(c, "iam.rbac.list-permissions", "permissions listed")
	response.RespondSuccess(c, perms, "ok")
}

// @Router /api/v1/admin/rbac/permissions [post]
// @Tags RBAC
// @Summary Create permission
// @Description Create permission
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
func (h *RbacHandler) CreatePermission(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req iam_reqdto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	perm := &entity.Permission{Name: req.Name, Description: req.Description}
	if err := h.svc.CreatePermission(ctx, perm); err != nil {
		logger.HandlerError(c, "iam.rbac.create-permission", err)
		response.RespondInternalError(c, "failed to create permission")
		return
	}
	logger.HandlerInfo(c, "iam.rbac.create-permission", "permission created")
	c.JSON(http.StatusCreated, gin.H{"permission": perm, "message": "permission created"})
}

// @Router /api/v1/admin/rbac/roles/:id/permissions [post]
// @Tags RBAC
// @Summary Assign permission
// @Description Assign permission
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
func (h *RbacHandler) AssignPermission(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req iam_reqdto.AssignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	if err := h.svc.AssignPermission(ctx, c.Param("id"), req.PermissionID); err != nil {
		logger.HandlerError(c, "iam.rbac.assign-permission", err)
		h.mapError(c, err)
		return
	}
	logger.HandlerInfo(c, "iam.rbac.assign-permission", "permission assigned — cache invalidated")
	response.RespondSuccess(c, nil, "permission assigned — cache invalidated")
}

// @Router /api/v1/admin/rbac/roles/:id/permissions/:perm_id [delete]
// @Tags RBAC
// @Summary Revoke permission
// @Description Revoke permission
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
func (h *RbacHandler) RevokePermission(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.svc.RevokePermission(ctx, c.Param("id"), c.Param("perm_id")); err != nil {
		logger.HandlerError(c, "iam.rbac.revoke-permission", err)
		h.mapError(c, err)
		return
	}
	logger.HandlerInfo(c, "iam.rbac.revoke-permission", "permission revoked — cache invalidated")
	response.RespondSuccess(c, nil, "permission revoked — cache invalidated")
}

// @Router /api/v1/admin/rbac/cache/invalidate [post]
// @Tags RBAC
// @Summary Invalidate cache
// @Description Invalidate cache
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
func (h *RbacHandler) InvalidateAll(c *gin.Context) {
	h.svc.InvalidateAll(c.Request.Context())
	logger.HandlerWarn(c, "iam.rbac.cache-invalidate", nil, "entire rbac cache flushed")
	response.RespondSuccess(c, nil, "rbac cache flushed")
}

// ── error mapping ─────────────────────────────────────────────────────────────

func (h *RbacHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, iam_errorx.ErrRoleNotFound):
		response.RespondNotFound(c, "role not found")
	case errors.Is(err, iam_errorx.ErrPermissionNotFound):
		response.RespondNotFound(c, "permission not found")
	case errors.Is(err, iam_errorx.ErrRoleAlreadyExists):
		response.RespondConflict(c, "role already exists")
	default:
		response.RespondInternalError(c, "rbac operation failed")
	}
}
