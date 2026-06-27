package user

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

func (h *Handler) GetMyProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	profile, err := h.service.GetMyProfile(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy thông tin profile thành công",
		"data":    profile,
	})
}

func (h *Handler) UpdateMyProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req UpdateUserProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Dữ liệu không hợp lệ",
			"error":   err.Error(),
		})
		return
	}

	if err := h.service.UpdateMyProfile(userID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cập nhật profile thành công",
	})
}
