package account

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

	accountGroup := r.Group("/accounts")

	accountGroup.Use(middleware.AuthMiddleware(cfg))
	{
		accountGroup.GET("", handler.GetMyAccounts)
		accountGroup.POST("", handler.CreateAccount)
	}
}
