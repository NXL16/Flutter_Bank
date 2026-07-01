package admin

import (
	"net/http"
	"strconv"

	"bank-service/internal/modules/transaction"

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

func (h *Handler) GetDashboard(c *gin.Context) {
	dashboard, err := h.service.GetDashboard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Không thể tải dữ liệu điều hành",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy dữ liệu điều hành thành công",
		"data":    dashboard,
	})
}

func (h *Handler) CreateStepUp(c *gin.Context) {
	var req StepUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Mã TOTP hoặc hành động không hợp lệ",
		})
		return
	}
	result, err := h.service.CreateStepUp(c.GetUint("user_id"), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Xác thực nâng cao thành công",
		"data":    result,
	})
}

func (h *Handler) GetTransactions(c *gin.Context) {
	limit := parseLimit(c.Query("limit"), 100)
	items, err := h.service.GetTransactions(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Không thể tải danh sách giao dịch",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy danh sách giao dịch thành công",
		"data":    items,
	})
}

func (h *Handler) GetAuditLogs(c *gin.Context) {
	limit := parseLimit(c.Query("limit"), 100)
	items, err := h.service.GetAuditLogs(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Không thể tải nhật ký quản trị",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy nhật ký quản trị thành công",
		"data":    items,
	})
}

func (h *Handler) GetAllUsers(c *gin.Context) {
	users, err := h.service.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Không thể lấy danh sách người dùng",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy danh sách người dùng thành công",
		"data":    users,
	})
}

func (h *Handler) GetUserByID(c *gin.Context) {
	userIDParam := c.Param("id")

	userID64, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID người dùng không hợp lệ",
		})
		return
	}

	user, err := h.service.GetUserByID(uint(userID64))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Không tìm thấy người dùng",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy thông tin người dùng thành công",
		"data":    user,
	})
}

func (h *Handler) LockUser(c *gin.Context) {
	userIDParam := c.Param("id")

	userID64, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID người dùng không hợp lệ",
		})
		return
	}
	if !h.authorizeStepUp(
		c,
		ActionLockUser,
		userActionBinding(ActionLockUser, uint(userID64)),
	) {
		return
	}

	if err := h.service.LockUser(
		c.GetUint("user_id"),
		uint(userID64),
		c.ClientIP(),
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Khóa tài khoản người dùng thành công",
	})
}

func (h *Handler) UnlockUser(c *gin.Context) {
	userIDParam := c.Param("id")

	userID64, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID người dùng không hợp lệ",
		})
		return
	}
	if !h.authorizeStepUp(
		c,
		ActionUnlockUser,
		userActionBinding(ActionUnlockUser, uint(userID64)),
	) {
		return
	}

	if err := h.service.UnlockUser(
		c.GetUint("user_id"),
		uint(userID64),
		c.ClientIP(),
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Mở khóa tài khoản người dùng thành công",
	})
}

func (h *Handler) GetUserAccounts(c *gin.Context) {
	userIDParam := c.Param("id")
	userID64, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID người dùng không hợp lệ",
		})
		return
	}

	accounts, err := h.service.GetUserAccounts(uint(userID64))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy danh sách tài khoản thành công",
		"data":    accounts,
	})
}

func (h *Handler) CreateAdmin(c *gin.Context) {
	role := c.GetString("role")
	if role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Chỉ có Super Admin mới có quyền thực hiện chức năng này",
		})
		return
	}

	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Dữ liệu không hợp lệ",
			"error":   err.Error(),
		})
		return
	}
	if !h.authorizeStepUp(c, ActionCreateAdmin, createAdminBinding(req)) {
		return
	}

	res, err := h.service.CreateAdmin(
		c.GetUint("user_id"),
		c.ClientIP(),
		req,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Tạo tài khoản Admin con thành công",
		"data":    res,
	})
}

func (h *Handler) Deposit(c *gin.Context) {
	adminUserID := c.GetUint("user_id")
	if adminUserID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Không xác định được danh tính quản trị viên",
		})
		return
	}

	var req transaction.DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Dữ liệu nạp tiền không hợp lệ",
			"error":   err.Error(),
		})
		return
	}
	req.IdempotencyKey = c.GetHeader("Idempotency-Key")
	if !h.authorizeStepUp(c, ActionDeposit, depositBinding(req)) {
		return
	}

	res, err := h.service.Deposit(adminUserID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Nạp tiền vào tài khoản thành công",
		"data":    res,
	})
}

func (h *Handler) authorizeStepUp(
	c *gin.Context,
	action string,
	binding string,
) bool {
	if err := h.service.AuthorizeStepUp(
		c.GetUint("user_id"),
		action,
		c.GetHeader("X-Admin-Step-Up"),
		binding,
	); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return false
	}
	return true
}

func (h *Handler) GetAccountTransactions(c *gin.Context) {
	accountIDParam := c.Param("account_id")
	accountID, err := strconv.ParseUint(accountIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ID tài khoản không hợp lệ",
		})
		return
	}

	transactions, err := h.service.GetAccountTransactions(uint(accountID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lấy lịch sử giao dịch thành công",
		"data":    transactions,
	})
}

func parseLimit(value string, fallback int) int {
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return fallback
	}
	if limit > 200 {
		return 200
	}
	return limit
}
