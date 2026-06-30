package admin

import (
	"bank-service/internal/infrastructure/totp"
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/auth"
	"bank-service/internal/modules/transaction"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo               *Repository
	accountService     *account.Service
	transactionService *transaction.Service
}

func NewService(repo *Repository, accountService *account.Service, transactionService *transaction.Service) *Service {
	return &Service{
		repo:               repo,
		accountService:     accountService,
		transactionService: transactionService,
	}
}

func (s *Service) GetAllUsers() ([]AdminUserResponse, error) {
	users, err := s.repo.FindAllUsers()
	if err != nil {
		return nil, err
	}

	response := make([]AdminUserResponse, 0)

	for _, user := range users {
		response = append(response, mapUserToAdminResponse(user))
	}

	return response, nil
}

func (s *Service) GetUserByID(userID uint) (*AdminUserResponse, error) {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return nil, err
	}

	return &AdminUserResponse{
		ID:         user.ID,
		FullName:   user.FullName,
		Phone:      user.Phone,
		Role:       user.Role,
		IsVerified: user.IsVerified,
		IsLocked:   user.IsLocked,
	}, nil
}

func (s *Service) LockUser(userID uint) error {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}

	if user.Role != "user" {
		return errors.New("Không thể khóa tài khoản quản trị viên")
	}

	if user.IsLocked {
		return errors.New("Tài khoản đã bị khóa")
	}

	return s.repo.LockUser(userID)
}

func (s *Service) UnlockUser(userID uint) error {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}

	if !user.IsLocked {
		return errors.New("Tài khoản chưa bị khóa")
	}

	return s.repo.UnlockUser(userID)
}

func mapUserToAdminResponse(user auth.User) AdminUserResponse {
	return AdminUserResponse{
		ID:         user.ID,
		FullName:   user.FullName,
		Phone:      user.Phone,
		Role:       user.Role,
		IsVerified: user.IsVerified,
		IsLocked:   user.IsLocked,
	}
}

func (s *Service) GetUserAccounts(
	userID uint,
) ([]account.AccountResponse, error) {
	return s.accountService.GetUserAccounts(userID)
}

func (s *Service) CreateAdmin(req CreateAdminRequest) (*CreateAdminResponse, error) {
	phone := "+84" + normalizePhone(req.Phone)
	existingUser, err := s.repo.FindUserByPhone(phone)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("Số điện thoại đã được sử dụng")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	totpSecret := totp.GenerateSecret()

	adminUser := &auth.User{
		FullName:     req.FullName,
		Phone:        phone,
		PasswordHash: string(hashedPassword),
		Role:         "admin",
		IsVerified:   true,
		IsLocked:     false,
		TOTPSecret:   totpSecret,
	}

	if err := s.repo.CreateAdminUser(adminUser); err != nil {
		return nil, err
	}

	if err := s.accountService.CreateDefaultPaymentAccount(adminUser.ID); err != nil {
		return nil, err
	}

	return &CreateAdminResponse{
		ID:         adminUser.ID,
		FullName:   adminUser.FullName,
		Phone:      adminUser.Phone,
		Role:       adminUser.Role,
		TOTPSecret: adminUser.TOTPSecret,
		CreatedAt:  adminUser.CreatedAt,
	}, nil
}

func normalizePhone(phone string) string {
	digits := make([]rune, 0, len(phone))
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			digits = append(digits, char)
		}
	}
	value := string(digits)
	if len(value) >= 2 && value[:2] == "84" {
		return value[2:]
	}
	if len(value) > 0 && value[0] == '0' {
		return value[1:]
	}
	return value
}

func (s *Service) Deposit(adminUserID uint, req transaction.DepositRequest) (*transaction.TransactionResponse, error) {
	return s.transactionService.Deposit(adminUserID, req)
}

func (s *Service) GetAccountTransactions(accountID uint) ([]transaction.TransactionResponse, error) {
	return s.transactionService.GetTransactionsByAccountID(accountID)
}
