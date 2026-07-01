package savings

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"bank-service/internal/modules/account"
	"bank-service/internal/modules/transaction"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	demandInterestRate                 = 0.50
	minimumEarlyWithdrawalAmount int64 = 100000
)

func (s *Service) WithdrawEarly(
	userID uint,
	accountNumber string,
	req EarlyWithdrawalRequest,
) (*EarlyWithdrawalResponse, error) {
	accountNumber = strings.TrimSpace(accountNumber)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if req.Amount < minimumEarlyWithdrawalAmount {
		return nil, errors.New("Số tiền rút trước hạn tối thiểu là 100.000 VND")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9._:-]{16,64}$`).
		MatchString(req.IdempotencyKey) {
		return nil, errors.New("Thiếu hoặc sai định dạng Idempotency-Key")
	}

	existingSavings, err := s.repo.FindSavingsByAccountNumber(
		userID,
		accountNumber,
	)
	if err != nil {
		return nil, err
	}
	if existingSavings == nil {
		return nil, errors.New("Không tìm thấy sổ tiết kiệm")
	}
	existingWithdrawal, err := s.repo.FindWithdrawalByIdempotencyKey(
		userID,
		existingSavings.AccountID,
		req.IdempotencyKey,
	)
	if err != nil {
		return nil, err
	}
	if existingWithdrawal != nil {
		response := mapEarlyWithdrawalResponse(
			accountNumber,
			*existingWithdrawal,
		)
		return &response, nil
	}
	if err := s.pinVerifier.VerifyTransactionPIN(
		userID,
		req.TransactionPIN,
	); err != nil {
		return nil, err
	}

	now := time.Now()
	var result *SavingsWithdrawal
	var pushBody string
	var referenceCode string

	err = s.repo.db.Transaction(func(tx *gorm.DB) error {
		var detail SavingsDetail
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ?", existingSavings.AccountID).
			First(&detail).Error
		if err != nil {
			return err
		}
		if detail.IsSettled {
			return errors.New("Sổ tiết kiệm đã được tất toán")
		}
		if !now.Before(detail.EndDate) {
			return errors.New("Sổ đã đến hạn và đang chờ hệ thống xử lý đáo hạn")
		}

		var savingsAccount account.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&savingsAccount, detail.AccountID).Error; err != nil {
			return err
		}
		if savingsAccount.UserID != userID ||
			savingsAccount.AccountType != "SAVINGS" ||
			savingsAccount.Status != "ACTIVE" {
			return errors.New("Sổ tiết kiệm không ở trạng thái có thể rút")
		}
		if savingsAccount.Balance != detail.OriginalPrincipal {
			return errors.New("Số dư sổ tiết kiệm không khớp tiền gốc")
		}
		if req.Amount > detail.OriginalPrincipal {
			return errors.New("Số tiền rút vượt quá tiền gốc còn lại")
		}
		remainingPrincipal := detail.OriginalPrincipal - req.Amount
		if remainingPrincipal > 0 &&
			remainingPrincipal < minimumSavingsAmount {
			return errors.New(
				"Số dư còn lại phải từ 5.000.000 VND; vui lòng giảm số tiền rút hoặc rút toàn bộ",
			)
		}

		var paymentAccount account.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND account_type = ?", userID, "PAYMENT").
			First(&paymentAccount).Error; err != nil {
			return fmt.Errorf("không tìm thấy tài khoản nhận tiền: %w", err)
		}

		accruedDays := elapsedSavingsDays(detail.StartDate, now)
		interest := calculateDemandInterest(
			req.Amount,
			demandInterestRate,
			accruedDays,
		)
		paymentBalance := paymentAccount.Balance
		var interestTransactionID *uint
		if interest > 0 {
			var fundingAccount account.Account
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where(
					"account_number = ? AND account_type = ?",
					systemFundingAccountNumber,
					"SYSTEM_FUNDING",
				).
				First(&fundingAccount).Error; err != nil {
				return fmt.Errorf(
					"không tìm thấy tài khoản nguồn chi lãi: %w",
					err,
				)
			}
			fundingNewBalance := fundingAccount.Balance - interest
			paymentBalance += interest
			interestReference := fmt.Sprintf(
				"SAVEARLYINT-%d-%d",
				detail.AccountID,
				now.UnixNano(),
			)
			interestKey := derivedInterestIdempotencyKey(req.IdempotencyKey)
			interestTransaction := &transaction.Transaction{
				ReferenceCode:     interestReference,
				InitiatorUserID:   &userID,
				IdempotencyKey:    &interestKey,
				SenderAccountID:   &fundingAccount.ID,
				ReceiverAccountID: paymentAccount.ID,
				Amount:            interest,
				Currency:          "VND",
				Type:              "SAVINGS_EARLY_INTEREST",
				Status:            "SUCCESS",
				Description: fmt.Sprintf(
					"Lãi không kỳ hạn khi rút sổ %s trước hạn",
					savingsAccount.AccountNumber,
				),
			}
			if err := tx.Create(interestTransaction).Error; err != nil {
				return err
			}
			if err := tx.Model(&fundingAccount).
				Update("balance", fundingNewBalance).Error; err != nil {
				return err
			}
			if err := tx.Model(&paymentAccount).
				Update("balance", paymentBalance).Error; err != nil {
				return err
			}
			if err := transaction.CreateDoubleEntry(
				tx,
				interestTransaction.ID,
				fundingAccount.ID,
				paymentAccount.ID,
				interest,
				"VND",
				fundingNewBalance,
				paymentBalance,
			); err != nil {
				return err
			}
			interestTransactionID = &interestTransaction.ID
		}

		referenceCode = fmt.Sprintf(
			"SAVEARLY-%d-%d",
			detail.AccountID,
			now.UnixNano(),
		)
		principalTransaction := &transaction.Transaction{
			ReferenceCode:     referenceCode,
			InitiatorUserID:   &userID,
			IdempotencyKey:    &req.IdempotencyKey,
			SenderAccountID:   &savingsAccount.ID,
			ReceiverAccountID: paymentAccount.ID,
			Amount:            req.Amount,
			Currency:          "VND",
			Type:              "SAVINGS_EARLY_WITHDRAWAL",
			Status:            "SUCCESS",
			Description: fmt.Sprintf(
				"Rút trước hạn từ sổ tiết kiệm %s",
				savingsAccount.AccountNumber,
			),
		}
		if err := tx.Create(principalTransaction).Error; err != nil {
			return err
		}
		paymentBalance += req.Amount
		if err := tx.Model(&paymentAccount).
			Update("balance", paymentBalance).Error; err != nil {
			return err
		}
		savingsAccountUpdates := map[string]any{
			"balance": remainingPrincipal,
		}
		if remainingPrincipal == 0 {
			savingsAccountUpdates["status"] = "CLOSED"
		}
		if err := tx.Model(&savingsAccount).
			Updates(savingsAccountUpdates).Error; err != nil {
			return err
		}
		if err := transaction.CreateDoubleEntry(
			tx,
			principalTransaction.ID,
			savingsAccount.ID,
			paymentAccount.ID,
			req.Amount,
			"VND",
			remainingPrincipal,
			paymentBalance,
		); err != nil {
			return err
		}

		isFullWithdrawal := remainingPrincipal == 0
		detailUpdates := map[string]any{}
		if isFullWithdrawal {
			detailUpdates["is_settled"] = true
			detailUpdates["closed_at"] = now
			detailUpdates["closure_reason"] = "EARLY_WITHDRAWAL"
		} else {
			detailUpdates["original_principal"] = remainingPrincipal
		}
		if err := tx.Model(&detail).Updates(detailUpdates).Error; err != nil {
			return err
		}

		withdrawal := &SavingsWithdrawal{
			AccountID:              detail.AccountID,
			UserID:                 userID,
			IdempotencyKey:         req.IdempotencyKey,
			Amount:                 req.Amount,
			DemandInterestRate:     demandInterestRate,
			AccruedDays:            accruedDays,
			Interest:               interest,
			RemainingPrincipal:     remainingPrincipal,
			IsFullWithdrawal:       isFullWithdrawal,
			PrincipalTransactionID: principalTransaction.ID,
			InterestTransactionID:  interestTransactionID,
			ProcessedAt:            now,
		}
		if err := tx.Create(withdrawal).Error; err != nil {
			return err
		}

		action := "rút một phần"
		if isFullWithdrawal {
			action = "tất toán trước hạn"
		}
		pushBody = fmt.Sprintf(
			"Sổ %s đã %s: gốc %d VND, lãi không kỳ hạn %d VND. Số dư tài khoản thanh toán mới: %d VND.",
			savingsAccount.AccountNumber,
			action,
			req.Amount,
			interest,
			paymentBalance,
		)
		if err := s.notiService.CreateNotification(
			tx,
			userID,
			"SAVINGS_WITHDRAWAL",
			"Rút tiết kiệm trước hạn",
			pushBody,
		); err != nil {
			return err
		}
		result = withdrawal
		return nil
	})
	if err != nil {
		existing, findErr := s.repo.FindWithdrawalByIdempotencyKey(
			userID,
			existingSavings.AccountID,
			req.IdempotencyKey,
		)
		if findErr == nil && existing != nil {
			response := mapEarlyWithdrawalResponse(accountNumber, *existing)
			return &response, nil
		}
		return nil, err
	}

	_ = s.notiService.SendPushToUser(
		userID,
		"Rút tiết kiệm trước hạn",
		pushBody,
		map[string]string{
			"type":           "SAVINGS_WITHDRAWAL",
			"reference_code": referenceCode,
			"transaction":    "SAVINGS_EARLY_WITHDRAWAL",
		},
	)
	response := mapEarlyWithdrawalResponse(accountNumber, *result)
	return &response, nil
}

func derivedInterestIdempotencyKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("ei:%x", hash[:16])
}

func elapsedSavingsDays(startDate time.Time, now time.Time) int {
	if !now.After(startDate) {
		return 0
	}
	return int(now.Sub(startDate).Hours() / 24)
}

func calculateDemandInterest(
	principal int64,
	rate float64,
	accruedDays int,
) int64 {
	if principal <= 0 || rate <= 0 || accruedDays <= 0 {
		return 0
	}
	return int64(math.Round(
		float64(principal) * rate / 100 * float64(accruedDays) / 365,
	))
}

func mapEarlyWithdrawalResponse(
	accountNumber string,
	withdrawal SavingsWithdrawal,
) EarlyWithdrawalResponse {
	return EarlyWithdrawalResponse{
		AccountNumber:      accountNumber,
		WithdrawnPrincipal: withdrawal.Amount,
		DemandInterestRate: withdrawal.DemandInterestRate,
		AccruedDays:        withdrawal.AccruedDays,
		Interest:           withdrawal.Interest,
		TotalReceived:      withdrawal.Amount + withdrawal.Interest,
		RemainingPrincipal: withdrawal.RemainingPrincipal,
		IsFullWithdrawal:   withdrawal.IsFullWithdrawal,
		ProcessedAt:        withdrawal.ProcessedAt,
	}
}
