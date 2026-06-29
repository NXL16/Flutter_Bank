package savings

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"bank-service/internal/modules/account"
	"bank-service/internal/modules/notification"
	"bank-service/internal/modules/transaction"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	repo        *Repository
	notiService *notification.Service
}

func NewService(repo *Repository, notiService *notification.Service) *Service {
	return &Service{
		repo:        repo,
		notiService: notiService,
	}
}

func (s *Service) OpenSavings(userID uint, req CreateSavingsRequest) (*SavingsResponse, error) {
	// 1. Kiểm tra số tiền tối thiểu
	if req.Amount < 5000000 {
		return nil, errors.New("số tiền gửi tiết kiệm tối thiểu là 5.000.000 VND")
	}

	var savingsResult *SavingsDetail
	var savingsAccountNum string

	// 2. Chạy database transaction
	err := s.repo.db.Transaction(func(tx *gorm.DB) error {
		// a. Tìm tài khoản PAYMENT của user để trừ tiền (và lock để tránh race condition)
		var paymentAccount account.Account
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND account_type = ?", userID, "PAYMENT").
			First(&paymentAccount).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("không tìm thấy tài khoản thanh toán PAYMENT của người dùng")
			}
			return err
		}

		if paymentAccount.Status != "ACTIVE" {
			return errors.New("tài khoản thanh toán của bạn đang bị khóa hoặc không hoạt động")
		}

		if paymentAccount.Balance < req.Amount {
			return errors.New("số dư tài khoản thanh toán không đủ để gửi tiết kiệm")
		}

		// b. Kiểm tra xem user đã có tài khoản SAVINGS nào chưa
		var existingSavings account.Account
		err = tx.Where("user_id = ? AND account_type = ?", userID, "SAVINGS").
			First(&existingSavings).Error
		if err == nil {
			return errors.New("người dùng đã sở hữu tài khoản tiết kiệm SAVINGS. Mỗi khách hàng chỉ được mở tối đa 1 sổ tiết kiệm.")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// c. Trừ tiền tài khoản PAYMENT
		paymentNewBalance := paymentAccount.Balance - req.Amount
		if err := tx.Model(&paymentAccount).Update("balance", paymentNewBalance).Error; err != nil {
			return err
		}

		// d. Sinh số tài khoản SAVINGS mới
		accountNumber, err := s.generateUniqueAccountNumber(tx, userID)
		if err != nil {
			return err
		}
		savingsAccountNum = accountNumber

		// e. Tạo tài khoản SAVINGS mới trong bảng accounts
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

		// f. Tạo chi tiết sổ tiết kiệm
		startDate := time.Now()
		endDate := startDate.AddDate(1, 0, 0) // Kỳ hạn 1 năm
		detail := &SavingsDetail{
			AccountID:         savingsAccount.ID,
			InterestRate:      8.50,
			TermMonths:        12,
			StartDate:         startDate,
			EndDate:           endDate,
			OriginalPrincipal: req.Amount,
			IsSettled:         false,
		}
		if err := tx.Create(detail).Error; err != nil {
			return err
		}

		// g. Ghi nhận giao dịch chuyển tiền sang tiết kiệm
		refCode := fmt.Sprintf("SAV%d", time.Now().UnixNano())
		desc := "Gửi tiết kiệm trực tuyến kỳ hạn 12 tháng lãi suất 8.5%/năm"
		newTx := &transaction.Transaction{
			ReferenceCode:     refCode,
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

		// h. Tạo thông báo biến động số dư cho tài khoản PAYMENT
		notiMsg := fmt.Sprintf("Tài khoản thanh toán của bạn đã bị trừ -%d VND để mở sổ tiết kiệm kỳ hạn 12 tháng. Số dư ví thanh toán mới: %d VND.", req.Amount, paymentNewBalance)
		if err := s.notiService.CreateNotification(tx, userID, "BALANCE_FLUCTUATION", "Biến động số dư (-)", notiMsg); err != nil {
			return err
		}

		savingsResult = detail
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &SavingsResponse{
		ID:                savingsResult.ID,
		AccountNumber:     savingsAccountNum,
		OriginalPrincipal: savingsResult.OriginalPrincipal,
		InterestRate:      savingsResult.InterestRate,
		TermMonths:        savingsResult.TermMonths,
		StartDate:         savingsResult.StartDate,
		EndDate:           savingsResult.EndDate,
		IsSettled:         savingsResult.IsSettled,
		Status:            "ACTIVE",
	}, nil
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
