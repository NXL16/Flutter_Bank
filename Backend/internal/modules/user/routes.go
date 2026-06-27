package user

import (
	"bank-service/internal/config"
	"bank-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	r *gin.RouterGroup,
	handler *Handler,
	cfg *config.Config,
) {

	userGroup := r.Group("/users")

	userGroup.Use(middleware.AuthMiddleware(cfg))
	{
		userGroup.GET("/me", handler.GetMyProfile)
		userGroup.PUT("/me", handler.UpdateMyProfile)
	}
}
