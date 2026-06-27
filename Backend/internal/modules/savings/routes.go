package savings

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
	savingsGroup := r.Group("/savings")
	savingsGroup.Use(middleware.AuthMiddleware(cfg))
	{
		savingsGroup.POST("", handler.OpenSavings)
	}
}
