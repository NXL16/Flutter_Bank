package auth

import (
	"bank-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes đăng ký các route thuộc auth module
func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	authGroup := r.Group("/auth")

	authGroup.POST("/register", handler.Register)
	authGroup.POST("/verify-register", handler.ConfirmRegister)
	authGroup.POST("/login", handler.Login)
	authGroup.POST("/confirm-login", handler.ConfirmLogin)
	authGroup.POST("/logout", handler.Logout)
	authGroup.POST("/refresh", handler.Refresh)
	authGroup.GET("/login/status", handler.GetLoginStatus)
	authGroup.GET("/device-verification/confirm", handler.ConfirmDeviceVerification)
	authGroup.GET("/device-verification/reject", handler.RejectDeviceVerification)
	protectedAuth := authGroup.Group("")
	protectedAuth.Use(middleware.AuthMiddleware(handler.service.cfg))
	{
		protectedAuth.PUT("/change-password", handler.ChangePassword)
	}
	authGroup.POST("/password-reset/request", handler.ForgotPassword)
	authGroup.POST("/password-reset/confirm", handler.ResetPassword)
}
