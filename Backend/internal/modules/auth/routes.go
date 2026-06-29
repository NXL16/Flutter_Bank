package auth

import (
	"bank-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes đăng ký các route thuộc auth module
func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	authGroup := r.Group("/auth")

	authGroup.POST("/register", handler.Register)
	authGroup.POST("/login", handler.Login)
	authGroup.POST("/confirm-login", handler.ConfirmLogin)
	authGroup.POST("/logout", handler.Logout)
	authGroup.POST("/refresh", handler.Refresh)
	protectedAuth := authGroup.Group("")
	protectedAuth.Use(middleware.AuthMiddleware(handler.service.cfg))
	{
		protectedAuth.PUT("/change-password", handler.ChangePassword)
	}
	authGroup.POST("/password-reset/confirm", handler.ResetPassword)
}
