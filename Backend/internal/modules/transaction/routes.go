package transaction

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
	transactionGroup := r.Group("/transactions")

	transactionGroup.Use(middleware.AuthMiddleware(cfg))
	{
		transactionGroup.GET("", handler.GetMyTransactions)
		transactionGroup.POST("/transfer", handler.Transfer)
		transactionGroup.GET("/:reference_code", handler.GetTransactionDetail)
	}
}
