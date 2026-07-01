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
		savingsGroup.GET("/products", handler.GetProducts)
		savingsGroup.GET("", handler.GetMySavings)
		savingsGroup.POST("", handler.OpenSavings)
		savingsGroup.POST("/:accountNumber/withdraw", handler.WithdrawEarly)
	}
}
