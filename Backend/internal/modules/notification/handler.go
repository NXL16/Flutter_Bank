package notification

import (
	"net/http"
	"strconv"

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

// GetMyNotifications trả về toàn bộ thông báo của người dùng đang đăng nhập
func (h *Handler) GetMyNotifications(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Không xác định được người dùng",
		})
		return
	}

	notifications, err := h.service.GetNotifications(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy danh sách thông báo thành công",
		"data":    notifications,
	})
}

// MarkRead đánh dấu đã đọc 1 thông báo
func (h *Handler) MarkRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Không xác định được người dùng",
		})
		return
	}

	idStr := c.Param("id")
	idVal, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Mã ID thông báo không hợp lệ",
		})
		return
	}

	err = h.service.MarkNotificationRead(uint(idVal), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Đã đánh dấu thông báo là đã đọc",
	})
}

// MarkAllRead đánh dấu đã đọc toàn bộ thông báo chưa đọc của user
func (h *Handler) MarkAllRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Không xác định được người dùng",
		})
		return
	}

	err := h.service.MarkAllNotificationsRead(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Đã đánh dấu đọc toàn bộ thông báo thành công",
	})
}
