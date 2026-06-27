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

// CreateAccount mở thêm account mới cho user
func (h *Handler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Dữ liệu không hợp lệ",
			"error":   err.Error(),
		})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Không xác định được người dùng",
		})
		return
	}

	role := c.GetString("role")
	if (req.AccountType == "SAVINGS" || req.AccountType == "CREDIT") && role != "admin" && role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Tài khoản Tiết kiệm (SAVINGS) và Tín dụng (CREDIT) chỉ có thể được mở bởi Quản trị viên",
		})
		return
	}

	account, err := h.service.CreateAccount(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Mở tài khoản thành công",
		"data":    account,
	})
}
