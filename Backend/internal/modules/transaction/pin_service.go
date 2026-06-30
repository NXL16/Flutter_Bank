package transaction

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	maxTransactionPINAttempts = 5
	transactionPINLockTime    = 15 * time.Minute
)

func (s *Service) GetTransactionPINStatus(userID uint) (*TransactionPINStatusResponse, error) {
	pin, err := s.repo.FindTransactionPIN(userID)
	if err != nil {
		return nil, err
	}
	return &TransactionPINStatusResponse{HasPIN: pin != nil}, nil
}

func (s *Service) SetupTransactionPIN(userID uint, req SetupTransactionPINRequest) error {
	req.PIN = strings.TrimSpace(req.PIN)
	req.ConfirmPIN = strings.TrimSpace(req.ConfirmPIN)
	if err := validateTransactionPIN(req.PIN); err != nil {
		return err
	}
	if req.PIN != req.ConfirmPIN {
		return errors.New("Mã PIN nhập lại không khớp")
	}

	existing, err := s.repo.FindTransactionPIN(userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("Mã PIN giao dịch đã được thiết lập")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.PIN), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("Không thể bảo vệ mã PIN giao dịch")
	}
	if err := s.repo.CreateTransactionPIN(&TransactionPIN{
		UserID:  userID,
		PINHash: string(hash),
	}); err != nil {
		return errors.New("Không thể tạo mã PIN giao dịch")
	}
	return nil
}

func (s *Service) VerifyTransactionPIN(userID uint, rawPIN string) error {
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(rawPIN) {
		return errors.New("Mã PIN giao dịch phải gồm đúng 6 chữ số")
	}

	pin, err := s.repo.FindTransactionPIN(userID)
	if err != nil {
		return err
	}
	if pin == nil {
		return errors.New("Bạn chưa thiết lập mã PIN giao dịch")
	}

	now := time.Now()
	if pin.LockedUntil != nil && now.Before(*pin.LockedUntil) {
		return fmt.Errorf(
			"Mã PIN đang bị khóa tạm thời, vui lòng thử lại sau %d phút",
			int(time.Until(*pin.LockedUntil).Minutes())+1,
		)
	}
	if pin.LockedUntil != nil {
		_ = s.repo.ResetTransactionPINFailures(userID)
		pin.FailedAttempts = 0
		pin.LockedUntil = nil
	}

	if bcrypt.CompareHashAndPassword([]byte(pin.PINHash), []byte(rawPIN)) != nil {
		attempts, lockedUntil, recordErr := s.repo.RecordTransactionPINFailure(
			userID,
			maxTransactionPINAttempts,
			transactionPINLockTime,
		)
		if recordErr != nil {
			return errors.New("Không thể xác minh mã PIN giao dịch")
		}
		if lockedUntil != nil {
			return errors.New("Mã PIN sai quá 5 lần, giao dịch bị khóa trong 15 phút")
		}
		return fmt.Errorf("Mã PIN không đúng, còn %d lần thử", maxTransactionPINAttempts-attempts)
	}

	if pin.FailedAttempts > 0 || pin.LockedUntil != nil {
		if err := s.repo.ResetTransactionPINFailures(userID); err != nil {
			return errors.New("Không thể hoàn tất xác minh mã PIN")
		}
	}
	return nil
}

func validateTransactionPIN(pin string) error {
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(pin) {
		return errors.New("Mã PIN phải gồm đúng 6 chữ số")
	}

	weakPINs := map[string]struct{}{
		"000000": {}, "111111": {}, "222222": {}, "333333": {},
		"444444": {}, "555555": {}, "666666": {}, "777777": {},
		"888888": {}, "999999": {}, "012345": {}, "123456": {},
		"234567": {}, "345678": {}, "456789": {}, "987654": {},
		"876543": {}, "765432": {}, "654321": {},
	}
	if _, weak := weakPINs[pin]; weak {
		return errors.New("Mã PIN quá dễ đoán, vui lòng chọn mã khác")
	}
	return nil
}
