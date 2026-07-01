package savings

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"bank-service/internal/modules/account"
	"bank-service/internal/modules/transaction"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maturityBatchSize = 200

type maturityPush struct {
	userID        uint
	title         string
	body          string
	referenceCode string
	outcome       string
}

// StartMaturityWorker quét ngay khi backend khởi động và lặp lại theo interval.
// Mỗi sổ vẫn được khóa và kiểm tra lại trong DB transaction nên an toàn khi
// nhiều instance backend cùng chạy worker.
func (s *Service) StartMaturityWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		run := func() {
			if err := s.ProcessDueSavings(ctx, time.Now()); err != nil &&
				!errors.Is(err, context.Canceled) {
				log.Printf("Xử lý đáo hạn tiết kiệm có lỗi: %v", err)
			}
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

// ProcessDueSavings được tách public để có thể chạy từ worker, job scheduler
// bên ngoài hoặc kiểm thử mà không phụ thuộc ticker.
func (s *Service) ProcessDueSavings(ctx context.Context, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	accountIDs, err := s.repo.FindDueSavingsAccountIDs(now, maturityBatchSize)
	if err != nil {
		return err
	}

	var processingErrors []error
	for _, accountID := range accountIDs {
		if err := ctx.Err(); err != nil {
			return err
		}
		push, processErr := s.processMaturity(ctx, accountID, now)
		if processErr != nil {
			processingErrors = append(
				processingErrors,
				fmt.Errorf("sổ %d: %w", accountID, processErr),
			)
			continue
		}
		if push == nil {
			continue
		}
		_ = s.notiService.SendPushToUser(
			push.userID,
			push.title,
			push.body,
			map[string]string{
				"type":           "SAVINGS_MATURITY",
				"reference_code": push.referenceCode,
				"transaction":    "SAVINGS_MATURITY",
				"outcome":        push.outcome,
			},
		)
	}
	return errors.Join(processingErrors...)
}

func (s *Service) processMaturity(
	ctx context.Context,
	accountID uint,
	now time.Time,
) (*maturityPush, error) {
	var push *maturityPush
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var detail SavingsDetail
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ?", accountID).
			First(&detail).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if detail.IsSettled || detail.EndDate.After(now) {
			return nil
		}

		var savingsAccount account.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&savingsAccount, detail.AccountID).Error; err != nil {
			return err
		}
		if savingsAccount.AccountType != "SAVINGS" ||
			savingsAccount.Status != "ACTIVE" {
			return errors.New("tài khoản tiết kiệm không ở trạng thái có thể đáo hạn")
		}
		if savingsAccount.Balance != detail.OriginalPrincipal {
			return errors.New("số dư sổ tiết kiệm không khớp tiền gốc")
		}

		var paymentAccount account.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND account_type = ?", savingsAccount.UserID, "PAYMENT").
			First(&paymentAccount).Error; err != nil {
			return fmt.Errorf("không tìm thấy tài khoản nhận tiền đáo hạn: %w", err)
		}

		var fundingAccount account.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_number = ? AND account_type = ?", systemFundingAccountNumber, "SYSTEM_FUNDING").
			First(&fundingAccount).Error; err != nil {
			return fmt.Errorf("không tìm thấy tài khoản nguồn chi lãi: %w", err)
		}

		cycleNumber := detail.RenewalCount + 1
		interest := calculateTermInterest(
			detail.OriginalPrincipal,
			detail.InterestRate,
			detail.TermMonths,
		)
		interestReference := fmt.Sprintf(
			"SAVINT-%d-%d",
			detail.AccountID,
			cycleNumber,
		)
		interestKey := fmt.Sprintf(
			"savings-interest:%d:%d",
			detail.AccountID,
			cycleNumber,
		)
		userID := savingsAccount.UserID
		fundingNewBalance := fundingAccount.Balance - interest
		paymentAfterInterest := paymentAccount.Balance + interest
		interestTransaction := &transaction.Transaction{
			ReferenceCode:     interestReference,
			InitiatorUserID:   &userID,
			IdempotencyKey:    &interestKey,
			SenderAccountID:   &fundingAccount.ID,
			ReceiverAccountID: paymentAccount.ID,
			Amount:            interest,
			Currency:          "VND",
			Type:              "SAVINGS_INTEREST",
			Status:            "SUCCESS",
			Description: fmt.Sprintf(
				"Trả lãi sổ tiết kiệm %s kỳ %d",
				savingsAccount.AccountNumber,
				cycleNumber,
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
			Update("balance", paymentAfterInterest).Error; err != nil {
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
			paymentAfterInterest,
		); err != nil {
			return err
		}

		event := &SavingsMaturityEvent{
			AccountID:             detail.AccountID,
			CycleNumber:           cycleNumber,
			Principal:             detail.OriginalPrincipal,
			Interest:              interest,
			InterestRate:          detail.InterestRate,
			TermMonths:            detail.TermMonths,
			PeriodStart:           detail.StartDate,
			PeriodEnd:             detail.EndDate,
			Instruction:           detail.MaturityInstruction,
			InterestTransactionID: interestTransaction.ID,
			ProcessedAt:           now,
		}

		title := "Tiết kiệm đã đáo hạn"
		referenceCode := interestReference
		finalPaymentBalance := paymentAfterInterest
		if detail.MaturityInstruction == "PAYOUT" {
			principalReference := fmt.Sprintf(
				"SAVMAT-%d-%d",
				detail.AccountID,
				cycleNumber,
			)
			principalKey := fmt.Sprintf(
				"savings-maturity:%d:%d",
				detail.AccountID,
				cycleNumber,
			)
			finalPaymentBalance += detail.OriginalPrincipal
			principalTransaction := &transaction.Transaction{
				ReferenceCode:     principalReference,
				InitiatorUserID:   &userID,
				IdempotencyKey:    &principalKey,
				SenderAccountID:   &savingsAccount.ID,
				ReceiverAccountID: paymentAccount.ID,
				Amount:            detail.OriginalPrincipal,
				Currency:          "VND",
				Type:              "SAVINGS_MATURITY",
				Status:            "SUCCESS",
				Description: fmt.Sprintf(
					"Tất toán tiền gốc sổ tiết kiệm %s",
					savingsAccount.AccountNumber,
				),
			}
			if err := tx.Create(principalTransaction).Error; err != nil {
				return err
			}
			if err := tx.Model(&savingsAccount).Updates(map[string]any{
				"balance": 0,
				"status":  "CLOSED",
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&paymentAccount).
				Update("balance", finalPaymentBalance).Error; err != nil {
				return err
			}
			if err := transaction.CreateDoubleEntry(
				tx,
				principalTransaction.ID,
				savingsAccount.ID,
				paymentAccount.ID,
				detail.OriginalPrincipal,
				"VND",
				0,
				finalPaymentBalance,
			); err != nil {
				return err
			}
			if err := tx.Model(&detail).Updates(map[string]any{
				"is_settled":      true,
				"renewal_count":   cycleNumber - 1,
				"last_matured_at": now,
				"closed_at":       now,
				"closure_reason":  "MATURITY_PAYOUT",
			}).Error; err != nil {
				return err
			}
			event.Outcome = "PAYOUT"
			event.PrincipalTransactionID = &principalTransaction.ID
			referenceCode = principalReference
		} else if detail.MaturityInstruction == "RENEW_PRINCIPAL" {
			nextRate, exists := savingsProducts[detail.TermMonths]
			if !exists {
				return errors.New("không còn biểu lãi suất cho kỳ hạn tái tục")
			}
			nextStartDate := detail.EndDate
			nextEndDate := nextStartDate.AddDate(0, detail.TermMonths, 0)
			if err := tx.Model(&detail).Updates(map[string]any{
				"interest_rate":   nextRate,
				"start_date":      nextStartDate,
				"end_date":        nextEndDate,
				"renewal_count":   cycleNumber,
				"last_matured_at": now,
			}).Error; err != nil {
				return err
			}
			event.Outcome = "RENEWED"
			title = "Sổ tiết kiệm đã tái tục"
		} else {
			return errors.New("chỉ thị đáo hạn không được hỗ trợ")
		}

		if err := tx.Create(event).Error; err != nil {
			return err
		}

		var body string
		if event.Outcome == "PAYOUT" {
			body = fmt.Sprintf(
				"Sổ %s đã tất toán. Gốc %d VND và lãi %d VND đã chuyển vào tài khoản thanh toán. Số dư mới: %d VND.",
				savingsAccount.AccountNumber,
				detail.OriginalPrincipal,
				interest,
				finalPaymentBalance,
			)
		} else {
			body = fmt.Sprintf(
				"Sổ %s đã tái tục tiền gốc %d VND. Tiền lãi %d VND đã chuyển vào tài khoản thanh toán. Lãi suất kỳ mới: %.2f%%/năm.",
				savingsAccount.AccountNumber,
				detail.OriginalPrincipal,
				interest,
				savingsProducts[detail.TermMonths],
			)
		}
		if err := s.notiService.CreateNotification(
			tx,
			userID,
			"SAVINGS_MATURITY",
			title,
			body,
		); err != nil {
			return err
		}
		push = &maturityPush{
			userID:        userID,
			title:         title,
			body:          body,
			referenceCode: referenceCode,
			outcome:       event.Outcome,
		}
		return nil
	})
	return push, err
}
