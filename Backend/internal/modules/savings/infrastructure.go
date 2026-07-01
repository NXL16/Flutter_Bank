package savings

import (
	"errors"
	"fmt"

	"bank-service/internal/modules/account"
	"bank-service/internal/modules/auth"

	"gorm.io/gorm"
)

const (
	systemSavingsPhone         = "+84000000000"
	systemFundingAccountNumber = "SYS-SAVINGS-INTEREST"
)

// EnsureSystemInfrastructure tạo tài khoản nội bộ dùng làm đối ứng sổ cái
// khi NF Bank chi trả lãi. Tài khoản bị khóa đăng nhập và không thuộc khách hàng.
func (s *Service) EnsureSystemInfrastructure() error {
	return s.repo.db.Transaction(func(tx *gorm.DB) error {
		var systemUser auth.User
		err := tx.Where("phone = ?", systemSavingsPhone).First(&systemUser).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			systemUser = auth.User{
				FullName:     "NF Bank Savings System",
				PasswordHash: "SYSTEM_ACCOUNT_NO_LOGIN",
				Phone:        systemSavingsPhone,
				Role:         "system",
				IsVerified:   true,
				IsLocked:     true,
			}
			if err := tx.Create(&systemUser).Error; err != nil {
				return fmt.Errorf("tạo người dùng hệ thống tiết kiệm: %w", err)
			}
		} else if err != nil {
			return err
		} else if systemUser.Role != "system" {
			return errors.New("số điện thoại dành cho hệ thống tiết kiệm đã bị sử dụng")
		}

		var fundingAccount account.Account
		err = tx.Where("account_number = ?", systemFundingAccountNumber).
			First(&fundingAccount).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fundingAccount = account.Account{
				UserID:        systemUser.ID,
				AccountNumber: systemFundingAccountNumber,
				AccountType:   "SYSTEM_FUNDING",
				Balance:       0,
				Currency:      "VND",
				Status:        "ACTIVE",
			}
			return tx.Create(&fundingAccount).Error
		}
		if err != nil {
			return err
		}
		if fundingAccount.UserID != systemUser.ID ||
			fundingAccount.AccountType != "SYSTEM_FUNDING" {
			return errors.New("tài khoản nguồn chi lãi tiết kiệm không hợp lệ")
		}
		return nil
	})
}
