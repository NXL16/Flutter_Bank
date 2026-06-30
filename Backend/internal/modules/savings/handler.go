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

func (h *Handler) GetProducts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy danh sách sản phẩm tiết kiệm thành công",
		"data":    h.service.GetProducts(),
	})
}

func (h *Handler) GetMySavings(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Không xác định được người dùng đăng nhập",
		})
		return
	}
	items, err := h.service.GetUserSavings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Không thể lấy danh sách sổ tiết kiệm",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy danh sách sổ tiết kiệm thành công",
		"data":    items,
	})
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
			"message": "Thông tin mở sổ hoặc mã PIN giao dịch không hợp lệ",
		})
		return
	}
	req.IdempotencyKey = c.GetHeader("Idempotency-Key")

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
