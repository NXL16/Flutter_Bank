package savings

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"time"

	"bank-service/internal/modules/account"
	"bank-service/internal/modules/notification"
	"bank-service/internal/modules/transaction"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const minimumSavingsAmount int64 = 5000000

var savingsProducts = map[int]float64{
	3:  6.20,
	6:  7.00,
	12: 8.50,
}

type Service struct {
	repo        *Repository
	notiService *notification.Service
	pinVerifier interface {
		VerifyTransactionPIN(userID uint, rawPIN string) error
	}
}

func NewService(
	repo *Repository,
	notiService *notification.Service,
	pinVerifier interface {
		VerifyTransactionPIN(userID uint, rawPIN string) error
	},
) *Service {
	return &Service{
		repo:        repo,
		notiService: notiService,
		pinVerifier: pinVerifier,
	}
}

func (s *Service) GetProducts() []SavingsProductResponse {
	terms := []int{3, 6, 12}
	products := make([]SavingsProductResponse, 0, len(terms))
	for _, term := range terms {
		products = append(products, SavingsProductResponse{
			TermMonths:    term,
			InterestRate:  savingsProducts[term],
			MinimumAmount: minimumSavingsAmount,
		})
	}
	return products
}

func (s *Service) GetUserSavings(userID uint) ([]SavingsResponse, error) {
	details, err := s.repo.FindSavingsByUserID(userID)
	if err != nil {
		return nil, err
	}
	response := make([]SavingsResponse, 0, len(details))
	for _, detail := range details {
		response = append(response, mapSavingsResponse(detail))
	}
	return response, nil
}

func (s *Service) OpenSavings(userID uint, req CreateSavingsRequest) (*SavingsResponse, error) {
	if req.Amount < minimumSavingsAmount {
		return nil, errors.New("Số tiền gửi tiết kiệm tối thiểu là 5.000.000 VND")
	}
	interestRate, validProduct := savingsProducts[req.TermMonths]
	if !validProduct {
		return nil, errors.New("Kỳ hạn tiết kiệm không hợp lệ")
	}
	req.MaturityInstruction = strings.ToUpper(strings.TrimSpace(req.MaturityInstruction))
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if !regexp.MustCompile(`^[A-Za-z0-9._:-]{16,64}$`).MatchString(req.IdempotencyKey) {
		return nil, errors.New("Thiếu hoặc sai định dạng Idempotency-Key")
	}
	if req.MaturityInstruction != "PAYOUT" &&
		req.MaturityInstruction != "RENEW_PRINCIPAL" {
		return nil, errors.New("Chỉ thị đáo hạn không hợp lệ")
	}
	existing, err := s.repo.FindSavingsByIdempotencyKey(userID, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		response := mapSavingsResponse(*existing)
		return &response, nil
	}
	if err := s.pinVerifier.VerifyTransactionPIN(userID, req.TransactionPIN); err != nil {
		return nil, err
	}

	var savingsResult *SavingsDetail
	var pushMessage string
	var referenceCode string

	err = s.repo.db.Transaction(func(tx *gorm.DB) error {
		var paymentAccount account.Account
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND account_type = ?", userID, "PAYMENT").
			First(&paymentAccount).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("Không tìm thấy tài khoản thanh toán PAYMENT của người dùng")
			}
			return err
		}

		if paymentAccount.Status != "ACTIVE" {
			return errors.New("Tài khoản thanh toán của bạn đang bị khóa hoặc không hoạt động")
		}

		if paymentAccount.Balance < req.Amount {
			return errors.New("Số dư tài khoản thanh toán không đủ để gửi tiết kiệm")
		}

		paymentNewBalance := paymentAccount.Balance - req.Amount
		if err := tx.Model(&paymentAccount).Update("balance", paymentNewBalance).Error; err != nil {
			return err
		}

		accountNumber, err := s.generateUniqueAccountNumber(tx, userID)
		if err != nil {
			return err
		}

		savingsAccount := &account.Account{
			UserID:        userID,
			AccountNumber: accountNumber,
			AccountType:   "SAVINGS",
			Balance:       req.Amount,
			Currency:      "VND",
			Status:        "ACTIVE",
		}
		if err := tx.Create(savingsAccount).Error; err != nil {
			return err
		}

		startDate := time.Now()
		endDate := startDate.AddDate(0, req.TermMonths, 0)
		detail := &SavingsDetail{
			AccountID:           savingsAccount.ID,
			Account:             *savingsAccount,
			InterestRate:        interestRate,
			TermMonths:          req.TermMonths,
			StartDate:           startDate,
			EndDate:             endDate,
			OriginalPrincipal:   req.Amount,
			MaturityInstruction: req.MaturityInstruction,
			IsSettled:           false,
		}
		if err := tx.Create(detail).Error; err != nil {
			return err
		}

		refCode := fmt.Sprintf("SAV%d", time.Now().UnixNano())
		initiatorUserID := userID
		idempotencyKey := req.IdempotencyKey
		desc := fmt.Sprintf(
			"Gửi tiết kiệm trực tuyến kỳ hạn %d tháng lãi suất %.2f%%/năm",
			req.TermMonths,
			interestRate,
		)
		newTx := &transaction.Transaction{
			ReferenceCode:     refCode,
			InitiatorUserID:   &initiatorUserID,
			IdempotencyKey:    &idempotencyKey,
			SenderAccountID:   &paymentAccount.ID,
			ReceiverAccountID: savingsAccount.ID,
			Amount:            req.Amount,
			Currency:          "VND",
			Type:              "SAVINGS_DEPOSIT",
			Status:            "SUCCESS",
			Description:       desc,
		}
		if err := tx.Create(newTx).Error; err != nil {
			return err
		}
		if err := transaction.CreateDoubleEntry(
			tx,
			newTx.ID,
			paymentAccount.ID,
			savingsAccount.ID,
			req.Amount,
			"VND",
			paymentNewBalance,
			req.Amount,
		); err != nil {
			return err
		}

		notiMsg := fmt.Sprintf(
			"Tài khoản thanh toán của bạn đã bị trừ -%d VND để mở sổ tiết kiệm kỳ hạn %d tháng. Số dư tài khoản thanh toán mới: %d VND.",
			req.Amount,
			req.TermMonths,
			paymentNewBalance,
		)
		if err := s.notiService.CreateNotification(tx, userID, "BALANCE_FLUCTUATION", "Biến động số dư (-)", notiMsg); err != nil {
			return err
		}

		savingsResult = detail
		pushMessage = notiMsg
		referenceCode = refCode
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Push chỉ được gửi sau khi transaction tài chính đã commit thành công.
	// Lỗi FCM không được phép làm rollback hoặc báo thất bại cho giao dịch đã hoàn tất.
	_ = s.notiService.SendPushToUser(
		userID,
		"Biến động số dư (-)",
		pushMessage,
		map[string]string{
			"type":           "BALANCE_FLUCTUATION",
			"reference_code": referenceCode,
			"transaction":    "SAVINGS_DEPOSIT",
		},
	)

	response := mapSavingsResponse(*savingsResult)
	return &response, nil
}

func mapSavingsResponse(detail SavingsDetail) SavingsResponse {
	expectedInterest := int64(math.Round(
		float64(detail.OriginalPrincipal) *
			detail.InterestRate / 100 *
			float64(detail.TermMonths) / 12,
	))
	return SavingsResponse{
		AccountNumber:       detail.Account.AccountNumber,
		OriginalPrincipal:   detail.OriginalPrincipal,
		InterestRate:        detail.InterestRate,
		TermMonths:          detail.TermMonths,
		StartDate:           detail.StartDate,
		EndDate:             detail.EndDate,
		ExpectedInterest:    expectedInterest,
		MaturityAmount:      detail.OriginalPrincipal + expectedInterest,
		MaturityInstruction: detail.MaturityInstruction,
		IsSettled:           detail.IsSettled,
		Status:              detail.Account.Status,
	}
}

func (s *Service) generateUniqueAccountNumber(tx *gorm.DB, userID uint) (string, error) {
	var role string
	err := tx.Table("users").Select("role").Where("id = ?", userID).Row().Scan(&role)
	if err != nil {
		return "", err
	}

	prefix := "9704" // Mặc định cho user
	if role == "super_admin" {
		prefix = "9999"
	} else if role == "admin" {
		prefix = "8888"
	}

	for {
		accountNumber, err := generateAccountNumber(prefix)
		if err != nil {
			return "", err
		}

		var count int64
		err = tx.Table("accounts").Where("account_number = ?", accountNumber).Count(&count).Error
		if err != nil {
			return "", err
		}

		if count == 0 {
			return accountNumber, nil
		}
	}
}

func generateAccountNumber(prefix string) (string, error) {
	number := prefix
	lengthNeeded := 12 - len(prefix)
	if lengthNeeded < 0 {
		lengthNeeded = 8
	}

	for i := 0; i < lengthNeeded; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		number += fmt.Sprintf("%d", n.Int64())
	}

	return number, nil
}
