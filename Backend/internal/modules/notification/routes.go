package notification

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
	notiGroup := r.Group("/notifications")
	notiGroup.Use(middleware.AuthMiddleware(cfg))
	{
		notiGroup.GET("", handler.GetMyNotifications)
		notiGroup.PATCH("/:id/read", handler.MarkRead)
		notiGroup.PATCH("/read-all", handler.MarkAllRead)
	}
}
