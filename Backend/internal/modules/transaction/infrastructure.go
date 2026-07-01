package transaction

import (
	"errors"
	"fmt"

	"bank-service/internal/modules/account"
	"bank-service/internal/modules/auth"

	"gorm.io/gorm"
)

const (
	systemOperationsPhone         = "+84000000001"
	systemOperationsAccountNumber = "SYS-CUSTOMER-FUNDING"
)

// EnsureOperationsInfrastructure tạo nguồn đối ứng cho nghiệp vụ cấp tiền.
// Admin chỉ là người phê duyệt; tiền không đi qua tài khoản cá nhân của Admin.
func (s *Service) EnsureOperationsInfrastructure() error {
	return s.repo.db.Transaction(func(tx *gorm.DB) error {
		var systemUser auth.User
		err := tx.Where("phone = ?", systemOperationsPhone).
			First(&systemUser).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			systemUser = auth.User{
				FullName:     "NF Bank Operations System",
				PasswordHash: "SYSTEM_ACCOUNT_NO_LOGIN",
				Phone:        systemOperationsPhone,
				Role:         "system",
				IsVerified:   true,
				IsLocked:     true,
			}
			if err := tx.Create(&systemUser).Error; err != nil {
				return fmt.Errorf("tạo người dùng hệ thống vận hành: %w", err)
			}
		} else if err != nil {
			return err
		} else if systemUser.Role != "system" {
			return errors.New("số điện thoại hệ thống vận hành đã bị sử dụng")
		}

		var fundingAccount account.Account
		err = tx.Where("account_number = ?", systemOperationsAccountNumber).
			First(&fundingAccount).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&account.Account{
				UserID:        systemUser.ID,
				AccountNumber: systemOperationsAccountNumber,
				AccountType:   "SYSTEM_FUNDING",
				Balance:       0,
				Currency:      "VND",
				Status:        "ACTIVE",
			}).Error
		}
		if err != nil {
			return err
		}
		if fundingAccount.UserID != systemUser.ID ||
			fundingAccount.AccountType != "SYSTEM_FUNDING" {
			return errors.New("tài khoản nguồn vận hành không hợp lệ")
		}
		return nil
	})
}
