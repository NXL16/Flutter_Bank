package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"bank-service/internal/config"
	"bank-service/internal/infrastructure/firebase"
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/notification"
	"bank-service/internal/modules/transaction"

	"gorm.io/gorm"
)

type Service struct {
	repo           *Repository
	firebaseClient *firebase.Client
	cfg            *config.Config
	notiService    *notification.Service
}

func NewService(repo *Repository, firebaseClient *firebase.Client, cfg *config.Config, notiService *notification.Service) *Service {
	return &Service{
		repo:           repo,
		firebaseClient: firebaseClient,
		cfg:            cfg,
		notiService:    notiService,
	}
}

// CreatePayment khởi tạo phiên giao dịch cho Merchant
func (s *Service) CreatePayment(req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	merchant, err := s.repo.FindMerchantByCodeAndKey(req.PartnerCode, req.AccessKey)
	if err != nil {
		return nil, err
	}
	if merchant == nil {
		return nil, errors.New("đối tác không tồn tại hoặc sai Access Key")
	}

	// Xác thực chữ ký số (Signature)
	rawStr := buildCreatePaymentRawString(req)
	expectedSignature := calculateHMACSHA256(rawStr, merchant.SecretKey)
	if req.Signature != expectedSignature {
		return nil, errors.New("chữ ký số (signature) không hợp lệ")
	}

	token := generateUUID()
	session := &PaymentSession{
		MerchantID:   merchant.ID,
		PaymentToken: token,
		Amount:       req.Amount,
		OrderID:      req.OrderID,
		RequestID:    req.RequestID,
		OrderInfo:    req.OrderInfo,
		RedirectURL:  req.RedirectURL,
		IpnURL:       req.IpnURL,
		ExtraData:    req.ExtraData,
		Status:       "PENDING",
		ExpiresAt:    time.Now().Add(15 * time.Minute), // Hạn dùng 15 phút
	}

	if err := s.repo.CreateSession(session); err != nil {
		return nil, err
	}

	// Xây dựng payUrl dựa trên cấu hình APP_URL
	host := s.cfg.AppURL
	if host == "" {
		host = "http://localhost:3000"
	}
	payURL := fmt.Sprintf("%s/payment-gateway?token=%s", host, token)

	res := CreatePaymentResponse{
		PartnerCode: req.PartnerCode,
		OrderID:     req.OrderID,
		RequestID:   req.RequestID,
		Amount:      req.Amount,
		PayURL:      payURL,
		ResultCode:  0,
		Message:     "Giao dịch khởi tạo thành công.",
	}

	resRawStr := buildCreatePaymentResponseRawString(res)
	res.Signature = calculateHMACSHA256(resRawStr, merchant.SecretKey)

	return &res, nil
}

// GetPaymentSession lấy thông tin hóa đơn hiển thị lên trang Checkout
func (s *Service) GetPaymentSession(token string) (*PaymentSessionResponse, error) {
	session, err := s.repo.FindSessionByToken(token)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, errors.New("phiên giao dịch không tồn tại hoặc đã hết hạn")
	}

	return &PaymentSessionResponse{
		PaymentToken: session.PaymentToken,
		Amount:       session.Amount,
		OrderID:      session.OrderID,
		OrderInfo:    session.OrderInfo,
		MerchantName: session.Merchant.MerchantName,
		RedirectURL:  session.RedirectURL,
		Status:       session.Status,
	}, nil
}

// ConfirmPayment xác nhận OTP và thực hiện chuyển tiền
func (s *Service) ConfirmPayment(userID uint, req ConfirmPaymentRequest) (*string, error) {
	// 1. Xác thực OTP
	userPhone, err := s.repo.GetUserPhone(userID)
	if err != nil {
		return nil, errors.New("không tìm thấy thông tin số điện thoại người dùng")
	}

	if req.IDToken != "" {
		verifiedPhone, err := s.firebaseClient.VerifyIDToken(req.IDToken)
		if err != nil {
			return nil, fmt.Errorf("xác thực OTP thất bại: %v", err)
		}
		if normalizePhone(verifiedPhone) != normalizePhone(userPhone) {
			return nil, errors.New("số điện thoại xác thực không khớp với tài khoản")
		}
	} else if s.cfg.AllowTestPaymentOTP && req.OTP == "123456" {
		// Chỉ được bật rõ ràng trong môi trường phát triển.
	} else {
		return nil, errors.New("yêu cầu xác thực OTP để tiếp tục giao dịch")
	}

	// 2. Tìm phiên giao dịch
	session, err := s.repo.FindSessionByToken(req.PaymentToken)
	if err != nil || session == nil {
		return nil, errors.New("phiên thanh toán không tồn tại")
	}

	if session.Status != "PENDING" {
		return nil, fmt.Errorf("giao dịch đã được xử lý (trạng thái: %s)", session.Status)
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.repo.UpdateSessionStatus(session.ID, "FAILED")
		return nil, errors.New("phiên thanh toán đã hết hạn")
	}

	// 3. Tài khoản của User thanh toán
	userAccount, err := s.repo.FindAccountByIDAndUserID(req.PaymentAccountID, userID)
	if err != nil || userAccount == nil {
		return nil, errors.New("tài khoản thanh toán không hợp lệ")
	}

	if userAccount.AccountType != "PAYMENT" {
		return nil, errors.New("chỉ hỗ trợ thanh toán từ tài khoản PAYMENT")
	}

	if userAccount.Status != "ACTIVE" {
		return nil, errors.New("tài khoản thanh toán của bạn đang bị khóa")
	}

	// 4. Tài khoản ví tích lũy của Merchant
	merchantAccount, err := s.repo.FindAccountByID(session.Merchant.PaymentAccountID)
	if err != nil || merchantAccount == nil {
		return nil, errors.New("tài khoản ví thụ hưởng của đối tác không hợp lệ")
	}

	if userAccount.Balance < session.Amount {
		return nil, errors.New("số dư tài khoản không đủ để thanh toán")
	}

	if userAccount.Currency != merchantAccount.Currency {
		return nil, errors.New("không hỗ trợ thanh toán khác loại tiền tệ")
	}

	var refCode string

	// 5. DB Transaction thực hiện chuyển tiền
	err = s.repo.WithTx(func(tx *gorm.DB) error {
		// Khóa tài khoản chống Race Condition theo thứ tự ID
		var lockedUser, lockedMerchant *account.Account
		var err error
		if userAccount.ID < merchantAccount.ID {
			lockedUser, err = s.repo.FindAccountByIDForUpdate(tx, userAccount.ID)
			if err != nil {
				return err
			}
			lockedMerchant, err = s.repo.FindAccountByIDForUpdate(tx, merchantAccount.ID)
			if err != nil {
				return err
			}
		} else {
			lockedMerchant, err = s.repo.FindAccountByIDForUpdate(tx, merchantAccount.ID)
			if err != nil {
				return err
			}
			lockedUser, err = s.repo.FindAccountByIDForUpdate(tx, userAccount.ID)
			if err != nil {
				return err
			}
		}

		if lockedUser.Balance < session.Amount {
			return errors.New("số dư tài khoản không đủ")
		}

		userNewBalance := lockedUser.Balance - session.Amount
		merchantNewBalance := lockedMerchant.Balance + session.Amount

		if err := s.repo.UpdateAccountBalance(tx, lockedUser.ID, userNewBalance); err != nil {
			return err
		}
		if err := s.repo.UpdateAccountBalance(tx, lockedMerchant.ID, merchantNewBalance); err != nil {
			return err
		}

		refCode = fmt.Sprintf("PAY%d", time.Now().UnixNano())
		description := fmt.Sprintf("Thanh toán hóa đơn: %s cho đối tác %s (Mã đơn hàng: %s)", session.OrderInfo, session.Merchant.MerchantName, session.OrderID)

		newTx := &transaction.Transaction{
			ReferenceCode:     refCode,
			SenderAccountID:   &lockedUser.ID,
			ReceiverAccountID: lockedMerchant.ID,
			Amount:            session.Amount,
			Currency:          lockedUser.Currency,
			Type:              "PAYMENT_GATEWAY",
			Status:            "SUCCESS",
			Description:       description,
		}

		if err := s.repo.CreateTransaction(tx, newTx); err != nil {
			return err
		}
		if err := transaction.CreateDoubleEntry(
			tx,
			newTx.ID,
			lockedUser.ID,
			lockedMerchant.ID,
			session.Amount,
			lockedUser.Currency,
			userNewBalance,
			merchantNewBalance,
		); err != nil {
			return err
		}

		// Tạo thông báo biến động số dư cho khách hàng (trừ tiền mua hàng)
		customerMsg := fmt.Sprintf("Tài khoản của bạn đã bị trừ -%d VND để thanh toán hóa đơn cho đối tác %s (Mã đơn hàng: %s). Số dư mới: %d VND.", session.Amount, session.Merchant.MerchantName, session.OrderID, userNewBalance)
		if err := s.notiService.CreateNotification(tx, userID, "PAYMENT_GATEWAY", "Thanh toán thành công", customerMsg); err != nil {
			return err
		}

		// Tạo thông báo biến động số dư cho đối tác/merchant (cộng tiền bán hàng)
		merchantMsg := fmt.Sprintf("Tài khoản ví của bạn đã được cộng +%d VND từ thanh toán hóa đơn %s của khách hàng. Số dư mới: %d VND.", session.Amount, session.OrderID, merchantNewBalance)
		if err := s.notiService.CreateNotification(tx, lockedMerchant.UserID, "PAYMENT_GATEWAY", "Nhận tiền thanh toán", merchantMsg); err != nil {
			return err
		}

		if err := s.repo.UpdateSessionSuccess(tx, session.ID, userID, refCode); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 6. Gửi Webhook (IPN) không đồng bộ
	go s.TriggerWebhook(session.ID)

	// 7. Tạo link chuyển hướng quay lại App Âm nhạc
	redirectURL := buildRedirectURL(session, session.Merchant.SecretKey, refCode)

	return &redirectURL, nil
}

// GetPaymentStatus đối soát trạng thái đơn hàng cho đối tác
func (s *Service) GetPaymentStatus(partnerCode, orderID, requestID, signature string) (*CreatePaymentResponse, error) {
	var merchant *Merchant
	// Tìm đối tác
	var m Merchant
	err := s.repo.db.Where("partner_code = ?", partnerCode).First(&m).Error
	if err != nil {
		return nil, errors.New("đối tác không tồn tại")
	}
	merchant = &m

	// Xác thực chữ ký đối soát gửi đến
	rawStr := buildStatusRequestRawString(orderID, partnerCode, requestID)
	expectedSignature := calculateHMACSHA256(rawStr, merchant.SecretKey)
	if signature != expectedSignature {
		return nil, errors.New("chữ ký số đối soát không hợp lệ")
	}

	// Tìm session
	var session PaymentSession
	err = s.repo.db.Preload("Merchant").Where("merchant_id = ? AND order_id = ?", merchant.ID, orderID).First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &CreatePaymentResponse{
				PartnerCode: partnerCode,
				OrderID:     orderID,
				RequestID:   requestID,
				ResultCode:  1000,
				Message:     "Giao dịch không tồn tại.",
			}, nil
		}
		return nil, err
	}

	resultCode := 0
	message := "Giao dịch thành công."
	if session.Status == "PENDING" {
		resultCode = 9000 // Mã chờ thanh toán
		message = "Giao dịch đang chờ thanh toán."
	} else if session.Status == "FAILED" {
		resultCode = 99
		message = "Giao dịch thất bại."
	}

	res := CreatePaymentResponse{
		PartnerCode: partnerCode,
		OrderID:     orderID,
		RequestID:   requestID,
		Amount:      session.Amount,
		ResultCode:  resultCode,
		Message:     message,
	}

	// Ký chữ ký phản hồi
	resRawStr := fmt.Sprintf("amount=%d&message=%s&orderId=%s&partnerCode=%s&requestId=%s&resultCode=%d",
		res.Amount, res.Message, res.OrderID, res.PartnerCode, res.RequestID, res.ResultCode)
	res.Signature = calculateHMACSHA256(resRawStr, merchant.SecretKey)

	return &res, nil
}

// TriggerWebhook bắn kết quả về IPN Url của đối tác
func (s *Service) TriggerWebhook(sessionID uint) {
	session, err := s.repo.FindSessionByID(sessionID)
	if err != nil || session == nil {
		fmt.Printf("⚠️ [IPN] Không tìm thấy phiên giao dịch ID: %d\n", sessionID)
		return
	}

	payload := IpnPayload{
		PartnerCode:  session.Merchant.PartnerCode,
		OrderID:      session.OrderID,
		RequestID:    session.RequestID,
		Amount:       session.Amount,
		OrderInfo:    session.OrderInfo,
		OrderType:    "nfbank_gateway",
		TransID:      session.ID,
		ResultCode:   0,
		Message:      "Giao dịch thành công",
		PayType:      "payment",
		ResponseTime: time.Now().UnixNano() / 1e6, // ms
		ExtraData:    session.ExtraData,
	}

	if session.Status != "SUCCESS" {
		payload.ResultCode = 99
		payload.Message = "Giao dịch thất bại"
	}

	rawStr := buildIpnRawString(payload)
	payload.Signature = calculateHMACSHA256(rawStr, session.Merchant.SecretKey)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("⚠️ [IPN] Marshalling JSON failed: %v\n", err)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	maxAttempts := 5
	backoff := []time.Duration{5 * time.Second, 15 * time.Second, 45 * time.Second, 2 * time.Minute, 5 * time.Minute}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		fmt.Printf("📣 [IPN] Đang gửi Webhook (lần %d) cho OrderID: %s sang URL: %s...\n", attempt+1, session.OrderID, session.IpnURL)

		req, err := http.NewRequest("POST", session.IpnURL, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("⚠️ [IPN] Create request failed: %v\n", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Printf("✅ [IPN] Gửi Webhook thành công cho OrderID: %s. Phản hồi: 200 OK\n", session.OrderID)
				return
			}
			fmt.Printf("⚠️ [IPN] Đối tác phản hồi lỗi HTTP: %d\n", resp.StatusCode)
		} else {
			fmt.Printf("⚠️ [IPN] Kết nối tới IPN URL thất bại: %v\n", err)
		}

		if attempt < maxAttempts-1 {
			wait := backoff[attempt]
			fmt.Printf("⏳ [IPN] Sẽ thử lại sau %v...\n", wait)
			time.Sleep(wait)
		}
	}

	fmt.Printf("❌ [IPN] Gửi Webhook thất bại hoàn toàn sau %d lần thử cho OrderID: %s\n", maxAttempts, session.OrderID)
}

// Helpers
func calculateHMACSHA256(rawString, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(rawString))
	return hex.EncodeToString(h.Sum(nil))
}

func buildCreatePaymentRawString(req CreatePaymentRequest) string {
	return fmt.Sprintf("accessKey=%s&amount=%d&extraData=%s&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s",
		req.AccessKey,
		req.Amount,
		req.ExtraData,
		req.IpnURL,
		req.OrderID,
		req.OrderInfo,
		req.PartnerCode,
		req.RedirectURL,
		req.RequestID,
	)
}

func buildCreatePaymentResponseRawString(res CreatePaymentResponse) string {
	return fmt.Sprintf("amount=%d&message=%s&orderId=%s&partnerCode=%s&payUrl=%s&requestId=%s&resultCode=%d",
		res.Amount,
		res.Message,
		res.OrderID,
		res.PartnerCode,
		res.PayURL,
		res.RequestID,
		res.ResultCode,
	)
}

func buildIpnRawString(payload IpnPayload) string {
	return fmt.Sprintf("amount=%d&extraData=%s&message=%s&orderId=%s&partnerCode=%s&requestId=%s&resultCode=%d&transId=%d",
		payload.Amount,
		payload.ExtraData,
		payload.Message,
		payload.OrderID,
		payload.PartnerCode,
		payload.RequestID,
		payload.ResultCode,
		payload.TransID,
	)
}

func buildStatusRequestRawString(orderID, partnerCode, requestID string) string {
	return fmt.Sprintf("orderId=%s&partnerCode=%s&requestId=%s", orderID, partnerCode, requestID)
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func normalizePhone(phone string) string {
	reg := regexp.MustCompile(`\D`)
	digits := reg.ReplaceAllString(phone, "")

	if len(digits) >= 11 && digits[:2] == "84" {
		return digits[2:]
	}
	if len(digits) > 0 && digits[:1] == "0" {
		return digits[1:]
	}
	return digits
}

func buildRedirectURL(session *PaymentSession, secretKey string, refCode string) string {
	resultCode := 0
	message := "Success"
	if session.Status != "SUCCESS" {
		resultCode = 99
		message = "Failed"
	}

	rawStr := fmt.Sprintf("amount=%d&extraData=%s&message=%s&orderId=%s&partnerCode=%s&requestId=%s&resultCode=%d&transId=%s",
		session.Amount,
		session.ExtraData,
		message,
		session.OrderID,
		session.Merchant.PartnerCode,
		session.RequestID,
		resultCode,
		refCode,
	)

	signature := calculateHMACSHA256(rawStr, secretKey)

	u, err := url.Parse(session.RedirectURL)
	if err != nil {
		return fmt.Sprintf("%s?resultCode=%d&orderId=%s", session.RedirectURL, resultCode, session.OrderID)
	}

	q := u.Query()
	q.Set("partnerCode", session.Merchant.PartnerCode)
	q.Set("orderId", session.OrderID)
	q.Set("requestId", session.RequestID)
	q.Set("amount", fmt.Sprintf("%d", session.Amount))
	q.Set("resultCode", fmt.Sprintf("%d", resultCode))
	q.Set("message", message)
	q.Set("transId", refCode)
	q.Set("extraData", session.ExtraData)
	q.Set("signature", signature)

	u.RawQuery = q.Encode()
	return u.String()
}
