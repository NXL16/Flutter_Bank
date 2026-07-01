package account

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"gorm.io/gorm"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) WithTransaction(tx *gorm.DB) *Service {
	return NewService(s.repo.withDB(tx))
}

// GetUserAccounts lấy danh sách account của user
func (s *Service) GetUserAccounts(
	userID uint,
) ([]AccountResponse, error) {

	accounts, err := s.repo.FindAccountsByUserID(userID)
	if err != nil {
		return nil, err
	}

	response := make([]AccountResponse, 0)

	for _, account := range accounts {
		response = append(response, AccountResponse{
			ID:            account.ID,
			AccountNumber: account.AccountNumber,
			AccountType:   account.AccountType,
			Balance:       account.Balance,
			Currency:      account.Currency,
			Status:        account.Status,
		})
	}

	return response, nil
}

// generateUniqueAccountNumber tạo số tài khoản unique
func (s *Service) generateUniqueAccountNumber(userID uint) (string, error) {
	// Truy vấn role của user
	role, err := s.repo.FindUserRoleByID(userID)
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

		existingAccount, err := s.repo.FindAccountByNumber(accountNumber)
		if err != nil {
			return "", err
		}

		if existingAccount == nil {
			return accountNumber, nil
		}
	}
}

// generateAccountNumber tạo random account number với prefix
func generateAccountNumber(prefix string) (string, error) {
	number := prefix

	// Đảm bảo tổng chiều dài là 12 ký tự (giống như cũ: 4 ký tự prefix + 8 ký tự random)
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

// CreateDefaultPaymentAccount tạo tài khoản PAYMENT mặc định
func (s *Service) CreateDefaultPaymentAccount(
	userID uint,
) error {

	accountNumber, err := s.generateUniqueAccountNumber(userID)
	if err != nil {
		return err
	}

	account := &Account{
		UserID:        userID,
		AccountNumber: accountNumber,
		AccountType:   "PAYMENT",
		Balance:       0,
		Currency:      "VND",
		Status:        "ACTIVE",
	}

	return s.repo.CreateAccount(account)
}
