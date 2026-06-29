package transaction

import (
	"errors"

	"gorm.io/gorm"
)

// CreateDoubleEntry ghi đúng một chân Nợ và một chân Có cho một giao dịch.
// Hàm phải được gọi trong cùng database transaction với cập nhật số dư.
func CreateDoubleEntry(
	tx *gorm.DB,
	transactionID uint,
	debitAccountID uint,
	creditAccountID uint,
	amount int64,
	currency string,
	debitBalanceAfter int64,
	creditBalanceAfter int64,
) error {
	if tx == nil || transactionID == 0 || debitAccountID == 0 || creditAccountID == 0 {
		return errors.New("thông tin bút toán không hợp lệ")
	}
	if debitAccountID == creditAccountID {
		return errors.New("tài khoản ghi nợ và ghi có phải khác nhau")
	}
	if amount <= 0 || currency == "" {
		return errors.New("số tiền hoặc loại tiền bút toán không hợp lệ")
	}

	entries := []LedgerEntry{
		{
			TransactionID: transactionID,
			AccountID:     debitAccountID,
			Direction:     "DEBIT",
			Amount:        amount,
			Currency:      currency,
			BalanceAfter:  debitBalanceAfter,
		},
		{
			TransactionID: transactionID,
			AccountID:     creditAccountID,
			Direction:     "CREDIT",
			Amount:        amount,
			Currency:      currency,
			BalanceAfter:  creditBalanceAfter,
		},
	}
	return tx.Create(&entries).Error
}
