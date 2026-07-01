package admin

import (
	"bank-service/internal/modules/auth"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

var errTOTPAlreadyUsed = errors.New("TOTP_ALREADY_USED")

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) withDB(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithTx(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

func (r *Repository) FindAllUsers() ([]auth.User, error) {
	var users []auth.User

	err := r.db.
		Preload("Profile").
		Where("role <> ?", "system").
		Order("created_at desc").
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *Repository) FindUserByID(
	userID uint,
) (*auth.User, error) {

	var user auth.User

	err := r.db.Preload("Profile").Where("role <> ?", "system").First(&user, userID).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) LockUser(
	actorUserID uint,
	userID uint,
	ipAddress string,
	summary string,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&auth.User{}).
			Where("id = ?", userID).
			Updates(map[string]interface{}{
				"is_locked":       true,
				"session_version": gorm.Expr("session_version + 1"),
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&auth.RefreshToken{}).
			Where("user_id = ? AND is_revoked = ?", userID, false).
			Update("is_revoked", true).Error; err != nil {
			return err
		}
		return tx.Create(&AuditLog{
			ActorUserID: actorUserID,
			Action:      "LOCK_USER",
			TargetType:  "USER",
			TargetID:    stringID(userID),
			Summary:     summary,
			IPAddress:   ipAddress,
		}).Error
	})
}

func (r *Repository) UnlockUser(
	actorUserID uint,
	userID uint,
	ipAddress string,
	summary string,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Model(&auth.User{}).
			Where("id = ?", userID).
			Update("is_locked", false).Error; err != nil {
			return err
		}
		return tx.Create(&AuditLog{
			ActorUserID: actorUserID,
			Action:      "UNLOCK_USER",
			TargetType:  "USER",
			TargetID:    stringID(userID),
			Summary:     summary,
			IPAddress:   ipAddress,
		}).Error
	})
}

func (r *Repository) CreateAdminUser(user *auth.User) error {
	return r.db.Create(user).Error
}

func (r *Repository) FindUserByPhone(phone string) (*auth.User, error) {
	var user auth.User
	err := r.db.Where("phone = ?", phone).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

type dashboardMetrics struct {
	CustomerCount         int64
	AdminCount            int64
	LockedCustomerCount   int64
	PaymentBalance        int64
	ActiveSavingsCount    int64
	ActiveSavingsBalance  int64
	TodayTransactionCount int64
	TodayTransactionValue int64
}

func (r *Repository) DashboardMetrics(startOfDay time.Time) (dashboardMetrics, error) {
	var result dashboardMetrics
	if err := r.db.Raw(`
		SELECT
			COALESCE(SUM(role = 'user'), 0) AS customer_count,
			COALESCE(SUM(role IN ('admin', 'super_admin')), 0) AS admin_count,
			COALESCE(SUM(role = 'user' AND is_locked = TRUE), 0) AS locked_customer_count
		FROM users
		WHERE role <> 'system'
	`).Scan(&result).Error; err != nil {
		return result, err
	}

	var accountMetrics struct {
		PaymentBalance       int64
		ActiveSavingsCount   int64
		ActiveSavingsBalance int64
	}
	if err := r.db.Raw(`
		SELECT
			COALESCE(SUM(CASE WHEN accounts.account_type = 'PAYMENT' THEN accounts.balance ELSE 0 END), 0) AS payment_balance,
			COALESCE(SUM(CASE WHEN accounts.account_type = 'SAVINGS' AND accounts.status = 'ACTIVE' THEN 1 ELSE 0 END), 0) AS active_savings_count,
			COALESCE(SUM(CASE WHEN accounts.account_type = 'SAVINGS' AND accounts.status = 'ACTIVE' THEN accounts.balance ELSE 0 END), 0) AS active_savings_balance
		FROM accounts
		JOIN users ON users.id = accounts.user_id
		WHERE users.role = 'user'
	`).Scan(&accountMetrics).Error; err != nil {
		return result, err
	}
	result.PaymentBalance = accountMetrics.PaymentBalance
	result.ActiveSavingsCount = accountMetrics.ActiveSavingsCount
	result.ActiveSavingsBalance = accountMetrics.ActiveSavingsBalance

	var transactionMetrics struct {
		Count int64
		Value int64
	}
	if err := r.db.Model(&struct {
		ID uint
	}{}).
		Table("transactions").
		Select("COUNT(*) AS count, COALESCE(SUM(amount), 0) AS value").
		Where("status = ? AND created_at >= ?", "SUCCESS", startOfDay).
		Scan(&transactionMetrics).Error; err != nil {
		return result, err
	}
	result.TodayTransactionCount = transactionMetrics.Count
	result.TodayTransactionValue = transactionMetrics.Value
	return result, nil
}

func (r *Repository) FindRecentUsers(limit int) ([]auth.User, error) {
	var users []auth.User
	err := r.db.
		Preload("Profile").
		Where("role <> ?", "system").
		Order("created_at desc").
		Limit(limit).
		Find(&users).Error
	return users, err
}

func (r *Repository) FindRecentTransactions(
	limit int,
) ([]AdminTransactionSummary, error) {
	var transactions []AdminTransactionSummary
	err := r.db.
		Table("transactions").
		Select("reference_code, type, amount, currency, status, description, created_at").
		Order("created_at desc").
		Limit(limit).
		Scan(&transactions).Error
	return transactions, err
}

func (r *Repository) FindRecentAuditLogs(
	limit int,
) ([]AuditLogResponse, error) {
	var logs []AuditLogResponse
	err := r.db.
		Table("audit_logs").
		Select(`
			audit_logs.id,
			audit_logs.actor_user_id,
			users.full_name AS actor_name,
			audit_logs.action,
			audit_logs.target_type,
			audit_logs.target_id,
			audit_logs.summary,
			audit_logs.ip_address,
			audit_logs.created_at
		`).
		Joins("LEFT JOIN users ON users.id = audit_logs.actor_user_id").
		Order("audit_logs.created_at desc").
		Limit(limit).
		Scan(&logs).Error
	return logs, err
}

func (r *Repository) CreateAuditLog(log *AuditLog) error {
	return r.db.Create(log).Error
}

func (r *Repository) RecordMFAFailure(
	actorUserID uint,
	now time.Time,
	maxAttempts int,
	lockDuration time.Duration,
) (*time.Time, error) {
	var lockedUntil *time.Time
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var user auth.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&user, actorUserID).Error; err != nil {
			return err
		}
		if user.TOTPLockedUntil != nil && user.TOTPLockedUntil.After(now) {
			lockedUntil = user.TOTPLockedUntil
			return nil
		}
		user.TOTPFailedAttempts++
		user.TOTPLockedUntil = nil
		if user.TOTPFailedAttempts >= maxAttempts {
			lock := now.Add(lockDuration)
			user.TOTPLockedUntil = &lock
			lockedUntil = &lock
		}
		return tx.Model(&user).Updates(map[string]interface{}{
			"totp_failed_attempts": user.TOTPFailedAttempts,
			"totp_locked_until":    user.TOTPLockedUntil,
		}).Error
	})
	return lockedUntil, err
}

func (r *Repository) CreateStepUpChallenge(
	challenge *StepUpChallenge,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var usageCount int64
		if err := tx.Model(&auth.TOTPUsage{}).
			Where(
				"user_id = ? AND time_step = ?",
				challenge.ActorUserID,
				challenge.TOTPTimeStep,
			).
			Count(&usageCount).Error; err != nil {
			return err
		}
		if usageCount > 0 {
			return errTOTPAlreadyUsed
		}
		if err := tx.Create(&auth.TOTPUsage{
			UserID:   challenge.ActorUserID,
			TimeStep: challenge.TOTPTimeStep,
			Purpose:  "ADMIN_STEP_UP_" + challenge.Action,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&auth.User{}).
			Where("id = ?", challenge.ActorUserID).
			Updates(map[string]interface{}{
				"totp_failed_attempts": 0,
				"totp_locked_until":    nil,
			}).Error; err != nil {
			return err
		}
		return tx.Create(challenge).Error
	})
}

func (r *Repository) HasTOTPUsage(
	actorUserID uint,
	timeStep int64,
) (bool, error) {
	var count int64
	err := r.db.Model(&auth.TOTPUsage{}).
		Where(
			"user_id = ? AND time_step = ?",
			actorUserID,
			timeStep,
		).
		Count(&count).Error
	return count > 0, err
}

func (r *Repository) ConsumeStepUpChallenge(
	actorUserID uint,
	action string,
	tokenHash string,
	bindingHash string,
	now time.Time,
) (bool, error) {
	consumed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var challenge StepUpChallenge
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(
				"actor_user_id = ? AND action = ? AND token_hash = ? AND binding_hash = ?",
				actorUserID,
				action,
				tokenHash,
				bindingHash,
			).
			First(&challenge).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if challenge.UsedAt != nil || !challenge.ExpiresAt.After(now) {
			return nil
		}
		if err := tx.Model(&challenge).Update("used_at", now).Error; err != nil {
			return err
		}
		consumed = true
		return nil
	})
	return consumed, err
}

func stringID(id uint) string {
	return fmt.Sprintf("%d", id)
}
