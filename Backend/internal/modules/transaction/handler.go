package transaction

import (
	"bank-service/internal/shared/response"
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

func (h *Handler) Transfer(c *gin.Context) {
	var req TransferRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "Không xác định được người dùng", nil)
		return
	}

	transaction, err := h.service.Transfer(userID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Chuyển tiền thành công", transaction)
}

func (h *Handler) GetMyTransactions(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "Không xác định được người dùng", nil)
		return
	}

	transactions, err := h.service.GetMyTransactions(userID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Lấy lịch sử giao dịch thành công", transactions)
}

func (h *Handler) GetTransactionDetail(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "Không xác định được người dùng", nil)
		return
	}

	referenceCode := c.Param("reference_code")

	transaction, err := h.service.GetTransactionDetail(
		userID,
		referenceCode,
	)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Lấy chi tiết giao dịch thành công", transaction)
}
