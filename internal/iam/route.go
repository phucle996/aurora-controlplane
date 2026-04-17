package iam

import "github.com/gin-gonic/gin"

// RegisterRoutes registers HTTP routes for the IAM module.
func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	// e.g. POST /api/v1/iam/login
	router.POST("/login", m.AuthHandler.Login)
}
