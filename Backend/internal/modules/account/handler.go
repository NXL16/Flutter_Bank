package account

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GetMyAccounts lấy danh sách account của user đang đăng nhập
func (h *Handler) GetMyAccounts(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Không xác định được người dùng",
		})
		return
	}

	accounts, err := h.service.GetUserAccounts(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Không thể lấy danh sách tài khoản",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy danh sách tài khoản thành công",
		"data":    accounts,
	})
}
