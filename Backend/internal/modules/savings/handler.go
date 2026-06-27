package savings

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

// OpenSavings xử lý yêu cầu mở sổ tiết kiệm trực tuyến
func (h *Handler) OpenSavings(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Không xác định được người dùng đăng nhập",
		})
		return
	}

	var req CreateSavingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Thông tin đầu vào không hợp lệ. Số tiền tối thiểu là 5.000.000 VND.",
		})
		return
	}

	res, err := h.service.OpenSavings(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Mở sổ tiết kiệm trực tuyến thành công",
		"data":    res,
	})
}
