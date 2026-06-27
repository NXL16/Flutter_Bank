package payment

import (
	"net/http"

	"bank-service/internal/shared/response"

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

// CreatePayment API xử lý yêu cầu khởi tạo đơn hàng từ Merchant
func (h *Handler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	res, err := h.service.CreatePayment(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetPaymentSession API lấy thông tin đơn hàng để hiển thị lên trang Checkout
func (h *Handler) GetPaymentSession(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		response.Error(c, http.StatusBadRequest, "Thiếu token phiên thanh toán", nil)
		return
	}

	res, err := h.service.GetPaymentSession(token)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Lấy thông tin phiên thanh toán thành công", res)
}

// ConfirmPayment API xử lý người dùng xác thực OTP và thực hiện thanh toán
func (h *Handler) ConfirmPayment(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "Không xác định được người dùng", nil)
		return
	}

	var req ConfirmPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	redirectURL, err := h.service.ConfirmPayment(userID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Thanh toán thành công, đang chuyển hướng...", gin.H{
		"redirect_url": redirectURL,
	})
}

// GetPaymentStatus API đối soát trạng thái giao dịch cho Merchant
func (h *Handler) GetPaymentStatus(c *gin.Context) {
	partnerCode := c.Query("partnerCode")
	orderID := c.Query("orderId")
	requestID := c.Query("requestId")
	signature := c.Query("signature")

	if partnerCode == "" || orderID == "" || requestID == "" || signature == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"resultCode": 400,
			"message":    "Thiếu tham số bắt buộc trong query",
		})
		return
	}

	res, err := h.service.GetPaymentStatus(partnerCode, orderID, requestID, signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resultCode": 400,
			"message":    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}
