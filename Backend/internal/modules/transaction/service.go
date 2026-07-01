package transaction

import (
	"bank-service/internal/config"
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/notification"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	repo        *Repository
	notiService *notification.Service
	cfg         *config.Config
}

func NewService(
	repo *Repository,
	notiService *notification.Service,
	cfg *config.Config,
) *Service {
	return &Service{
		repo:        repo,
		notiService: notiService,
		cfg:         cfg,
	}
}

func (s *Service) Transfer(
	userID uint,
	req TransferRequest,
) (*TransactionResponse, error) {
	req.ReceiverAccountNumber = strings.TrimSpace(req.ReceiverAccountNumber)
	req.Description = strings.TrimSpace(req.Description)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)

	if req.Description == "" {
		senderName, err := s.repo.GetUserFullName(userID)
		if err != nil {
			return nil, errors.New("Không thể lấy tên người chuyển khoản")
		}
		req.Description = defaultTransferDescription(senderName)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9._:-]{16,64}$`).MatchString(req.IdempotencyKey) {
		return nil, errors.New("Thiếu hoặc sai định dạng Idempotency-Key")
	}
	if len([]rune(req.Description)) > 140 {
		return nil, errors.New("Nội dung chuyển tiền tối đa 140 ký tự")
	}
	if req.Amount < s.cfg.TransferMinAmount {
		return nil, fmt.Errorf("Số tiền chuyển tối thiểu là %d VND", s.cfg.TransferMinAmount)
	}
	if req.Amount > s.cfg.TransferMaxAmount {
		return nil, fmt.Errorf("Số tiền vượt hạn mức mỗi giao dịch %d VND", s.cfg.TransferMaxAmount)
	}

	existing, err := s.repo.FindTransactionByIdempotencyKey(userID, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return mapTransactionResponse(existing), nil
	}

	if err := s.VerifyTransactionPIN(userID, req.TransactionPIN); err != nil {
		return nil, err
	}

	var transactionResult *Transaction
	var senderUserID, receiverUserID uint
	var senderPushMessage, receiverPushMessage string

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		// 1. Tìm sender account (chưa lock) để lấy ID
		senderAccount, err := s.repo.FindPaymentAccountByUserID(userID)
		if err != nil {
			return err
		}
		if senderAccount == nil {
			return errors.New("Không tìm thấy tài khoản gửi")
		}

		// 2. Tìm receiver account (chưa lock) để lấy ID
		receiverAccount, err := s.repo.FindAccountByNumber(req.ReceiverAccountNumber)
		if err != nil {
			return err
		}
		if receiverAccount == nil {
			return errors.New("Không tìm thấy tài khoản nhận")
		}

		if senderAccount.ID == receiverAccount.ID {
			return errors.New("Không thể chuyển tiền cho chính tài khoản của mình")
		}

		// 3. Thực hiện khóa theo thứ tự ID tăng dần để tránh Deadlock
		var lockedSender, lockedReceiver *account.Account
		if senderAccount.ID < receiverAccount.ID {
			// Khóa sender trước, receiver sau
			lockedSender, err = s.repo.FindAccountByIDForUpdate(tx, senderAccount.ID)
			if err != nil {
				return err
			}
			lockedReceiver, err = s.repo.FindAccountByIDForUpdate(tx, receiverAccount.ID)
			if err != nil {
				return err
			}
		} else {
			// Khóa receiver trước, sender sau
			lockedReceiver, err = s.repo.FindAccountByIDForUpdate(tx, receiverAccount.ID)
			if err != nil {
				return err
			}
			lockedSender, err = s.repo.FindAccountByIDForUpdate(tx, senderAccount.ID)
			if err != nil {
				return err
			}
		}

		if lockedSender == nil {
			return errors.New("Không tìm thấy tài khoản gửi")
		}
		if lockedReceiver == nil {
			return errors.New("Không tìm thấy tài khoản nhận")
		}

		if lockedSender.Status != "ACTIVE" {
			return errors.New("Tài khoản gửi không hoạt động")
		}

		if lockedReceiver.Status != "ACTIVE" {
			return errors.New("Tài khoản nhận không hoạt động")
		}

		if lockedSender.Currency != lockedReceiver.Currency {
			return errors.New("Không thể chuyển tiền khác loại tiền tệ")
		}

		if lockedSender.Balance < req.Amount {
			return errors.New("Số dư không đủ")
		}

		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		usedToday, err := s.repo.SumSuccessfulOutgoingTransfers(
			tx,
			lockedSender.ID,
			startOfDay,
			startOfDay.AddDate(0, 0, 1),
		)
		if err != nil {
			return err
		}
		if usedToday+req.Amount > s.cfg.DailyTransferLimit {
			return fmt.Errorf("Giao dịch vượt hạn mức chuyển tiền trong ngày %d VND", s.cfg.DailyTransferLimit)
		}

		senderNewBalance := lockedSender.Balance - req.Amount
		receiverNewBalance := lockedReceiver.Balance + req.Amount

		if err := s.repo.UpdateAccountBalance(
			tx,
			lockedSender.ID,
			senderNewBalance,
		); err != nil {
			return err
		}

		if err := s.repo.UpdateAccountBalance(
			tx,
			lockedReceiver.ID,
			receiverNewBalance,
		); err != nil {
			return err
		}

		senderID := lockedSender.ID
		initiatorUserID := userID
		idempotencyKey := req.IdempotencyKey
		newTransaction := &Transaction{
			ReferenceCode:     generateReferenceCode(),
			InitiatorUserID:   &initiatorUserID,
			IdempotencyKey:    &idempotencyKey,
			SenderAccountID:   &senderID,
			ReceiverAccountID: lockedReceiver.ID,
			Amount:            req.Amount,
			Currency:          lockedSender.Currency,
			Type:              "TRANSFER",
			Status:            "SUCCESS",
			Description:       req.Description,
		}

		if err := s.repo.CreateTransaction(tx, newTransaction); err != nil {
			return err
		}
		if err := CreateDoubleEntry(
			tx,
			newTransaction.ID,
			lockedSender.ID,
			lockedReceiver.ID,
			req.Amount,
			lockedSender.Currency,
			senderNewBalance,
			receiverNewBalance,
		); err != nil {
			return err
		}

		// Tạo thông báo biến động số dư cho người gửi
		senderMsg := fmt.Sprintf("Tài khoản của bạn đã bị trừ -%d VND chuyển khoản đến số tài khoản %s. Số dư mới: %d VND. Nội dung: %s", req.Amount, lockedReceiver.AccountNumber, senderNewBalance, req.Description)
		if err := s.notiService.CreateNotification(tx, lockedSender.UserID, "BALANCE_FLUCTUATION", "Biến động số dư (-)", senderMsg); err != nil {
			return err
		}

		// Tạo thông báo biến động số dư cho người nhận
		receiverMsg := fmt.Sprintf("Tài khoản của bạn đã được cộng +%d VND từ số tài khoản %s. Số dư mới: %d VND. Nội dung: %s", req.Amount, lockedSender.AccountNumber, receiverNewBalance, req.Description)
		if err := s.notiService.CreateNotification(tx, lockedReceiver.UserID, "BALANCE_FLUCTUATION", "Biến động số dư (+)", receiverMsg); err != nil {
			return err
		}

		transactionResult = newTransaction
		senderUserID = lockedSender.UserID
		receiverUserID = lockedReceiver.UserID
		senderPushMessage = senderMsg
		receiverPushMessage = receiverMsg

		return nil
	})

	if err != nil {
		existing, findErr := s.repo.FindTransactionByIdempotencyKey(userID, req.IdempotencyKey)
		if findErr == nil && existing != nil {
			return mapTransactionResponse(existing), nil
		}
		return nil, err
	}

	pushData := map[string]string{
		"type":           "BALANCE_FLUCTUATION",
		"reference_code": transactionResult.ReferenceCode,
	}
	_ = s.notiService.SendPushToUser(
		senderUserID,
		"Biến động số dư (-)",
		senderPushMessage,
		pushData,
	)
	_ = s.notiService.SendPushToUser(
		receiverUserID,
		"Biến động số dư (+)",
		receiverPushMessage,
		pushData,
	)

	return mapTransactionResponse(transactionResult), nil
}

func mapTransactionResponse(transactionResult *Transaction) *TransactionResponse {
	return &TransactionResponse{
		ID:                transactionResult.ID,
		ReferenceCode:     transactionResult.ReferenceCode,
		SenderAccountID:   transactionResult.SenderAccountID,
		ReceiverAccountID: transactionResult.ReceiverAccountID,
		Amount:            transactionResult.Amount,
		Currency:          transactionResult.Currency,
		Type:              transactionResult.Type,
		Status:            transactionResult.Status,
		Description:       transactionResult.Description,
		CreatedAt:         transactionResult.CreatedAt,
	}
}

func (s *Service) ResolveAccount(
	userID uint,
	accountNumber string,
) (*AccountResolutionResponse, error) {
	accountNumber = strings.TrimSpace(accountNumber)
	if !regexp.MustCompile(`^[0-9]{12}$`).MatchString(accountNumber) {
		return nil, errors.New("Số tài khoản phải gồm 12 chữ số")
	}

	ownAccount, err := s.repo.FindPaymentAccountByUserID(userID)
	if err != nil {
		return nil, err
	}
	if ownAccount != nil && ownAccount.AccountNumber == accountNumber {
		return nil, errors.New("Không thể chuyển tiền cho chính tài khoản của mình")
	}

	result, err := s.repo.ResolveActivePaymentAccount(accountNumber)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("Không tìm thấy tài khoản nhận đang hoạt động")
	}
	return result, nil
}

func generateReferenceCode() string {
	return fmt.Sprintf("TRX%d", time.Now().UnixNano())
}

func (s *Service) GetMyTransactions(
	userID uint,
) ([]TransactionResponse, error) {

	paymentAccount, err := s.repo.FindPaymentAccountByUserID(userID)
	if err != nil {
		return nil, err
	}

	if paymentAccount == nil {
		return nil, errors.New("Không tìm thấy tài khoản PAYMENT")
	}

	transactions, err := s.repo.FindTransactionViewsByAccountID(paymentAccount.ID)
	if err != nil {
		return nil, err
	}

	response := make([]TransactionResponse, 0)

	for _, transaction := range transactions {
		response = append(response, mapTransactionView(transaction))
	}

	return response, nil
}

func (s *Service) GetTransactionDetail(
	userID uint,
	referenceCode string,
) (*TransactionResponse, error) {

	paymentAccount, err := s.repo.FindPaymentAccountByUserID(userID)
	if err != nil {
		return nil, err
	}

	if paymentAccount == nil {
		return nil, errors.New("Không tìm thấy tài khoản PAYMENT")
	}

	transaction, err := s.repo.FindTransactionViewByReferenceCode(paymentAccount.ID, referenceCode)
	if err != nil {
		return nil, err
	}

	if transaction == nil {
		return nil, errors.New("Không tìm thấy giao dịch")
	}

	result := mapTransactionView(*transaction)
	return &result, nil
}

func (s *Service) Deposit(adminUserID uint, req DepositRequest) (*TransactionResponse, error) {
	req.ReceiverAccountNumber = strings.TrimSpace(req.ReceiverAccountNumber)
	req.Description = strings.TrimSpace(req.Description)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if !regexp.MustCompile(`^[A-Za-z0-9._:-]{16,64}$`).MatchString(req.IdempotencyKey) {
		return nil, errors.New("Thiếu hoặc sai định dạng Idempotency-Key")
	}
	if !regexp.MustCompile(`^[0-9]{12}$`).MatchString(req.ReceiverAccountNumber) {
		return nil, errors.New("Số tài khoản nhận phải gồm 12 chữ số")
	}
	if req.Amount < s.cfg.TransferMinAmount {
		return nil, fmt.Errorf(
			"Số tiền nạp tối thiểu là %d VND",
			s.cfg.TransferMinAmount,
		)
	}
	if req.Amount > s.cfg.AdminDepositMaxAmount {
		return nil, fmt.Errorf(
			"Số tiền nạp vượt hạn mức mỗi giao dịch %d VND",
			s.cfg.AdminDepositMaxAmount,
		)
	}
	if len([]rune(req.Description)) > 140 {
		return nil, errors.New("Nội dung nạp tiền tối đa 140 ký tự")
	}
	existing, err := s.repo.FindTransactionByIdempotencyKey(
		adminUserID,
		req.IdempotencyKey,
	)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return mapTransactionResponse(existing), nil
	}

	// Admin chỉ là người khởi tạo/phê duyệt nghiệp vụ. Nguồn hạch toán luôn
	// là tài khoản đối ứng hệ thống, tuyệt đối không dùng tài khoản cá nhân
	// của Admin dù dữ liệu cũ có tồn tại tài khoản PAYMENT.
	fundingAccount, err := s.repo.FindAccountByNumber(
		systemOperationsAccountNumber,
	)
	if err != nil {
		return nil, err
	}
	if fundingAccount == nil ||
		fundingAccount.AccountType != "SYSTEM_FUNDING" {
		return nil, errors.New("Nguồn cấp tiền hệ thống chưa sẵn sàng")
	}

	receiverAccount, err := s.repo.FindAccountByNumber(req.ReceiverAccountNumber)
	if err != nil {
		return nil, err
	}
	if receiverAccount == nil {
		return nil, errors.New("Không tìm thấy số tài khoản người nhận")
	}
	if receiverAccount.AccountType != "PAYMENT" {
		return nil, errors.New("Chỉ có thể nạp vào tài khoản thanh toán")
	}
	receiverRole, err := s.repo.GetUserRole(receiverAccount.UserID)
	if err != nil {
		return nil, err
	}
	if receiverRole != "user" {
		return nil, errors.New("Chỉ có thể nạp tiền cho tài khoản khách hàng")
	}

	var adminUser struct {
		FullName string
	}
	err = s.repo.db.Table("users").Where("id = ?", adminUserID).First(&adminUser).Error
	if err != nil {
		return nil, errors.New("Không thể tìm thấy thông tin định danh của admin thực hiện")
	}

	var transactionResult *Transaction
	var receiverPushMessage string

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		// Khóa 2 tài khoản theo thứ tự ID để chống deadlock
		var lockedAdmin, lockedReceiver *account.Account
		if fundingAccount.ID < receiverAccount.ID {
			lockedAdmin, err = s.repo.FindAccountByIDForUpdate(tx, fundingAccount.ID)
			if err != nil {
				return err
			}
			lockedReceiver, err = s.repo.FindAccountByIDForUpdate(tx, receiverAccount.ID)
			if err != nil {
				return err
			}
		} else {
			lockedReceiver, err = s.repo.FindAccountByIDForUpdate(tx, receiverAccount.ID)
			if err != nil {
				return err
			}
			lockedAdmin, err = s.repo.FindAccountByIDForUpdate(tx, fundingAccount.ID)
			if err != nil {
				return err
			}
		}

		if lockedAdmin == nil || lockedReceiver == nil {
			return errors.New("Không tìm thấy thông tin tài khoản")
		}
		if lockedAdmin.Status != "ACTIVE" {
			return errors.New("Tài khoản nguồn hoặc nguồn cấp tiền hệ thống không hoạt động")
		}

		if lockedReceiver.Status != "ACTIVE" {
			return errors.New("Tài khoản người nhận đang bị khóa hoặc không hoạt động")
		}

		if lockedAdmin.Currency != lockedReceiver.Currency {
			return errors.New("Không thể chuyển tiền khác loại tiền tệ")
		}

		// Tài khoản funding là tài khoản đối ứng hệ thống, không phải ví cá nhân
		// của Admin. Số dư âm thể hiện nghĩa vụ cấp vốn của ngân hàng.
		adminNewBalance := lockedAdmin.Balance - req.Amount
		receiverNewBalance := lockedReceiver.Balance + req.Amount

		if err := s.repo.UpdateAccountBalance(tx, lockedAdmin.ID, adminNewBalance); err != nil {
			return err
		}

		if err := s.repo.UpdateAccountBalance(tx, lockedReceiver.ID, receiverNewBalance); err != nil {
			return err
		}

		// Định dạng Description chứa thông tin đối soát
		formattedDesc := fmt.Sprintf("Nạp tiền từ Admin: %s (STK: %s)", adminUser.FullName, lockedAdmin.AccountNumber)
		if req.Description != "" {
			formattedDesc = fmt.Sprintf("%s - %s", formattedDesc, req.Description)
		}

		newTransaction := &Transaction{
			ReferenceCode:     generateReferenceCode(),
			InitiatorUserID:   &adminUserID,
			IdempotencyKey:    &req.IdempotencyKey,
			SenderAccountID:   &lockedAdmin.ID,
			ReceiverAccountID: lockedReceiver.ID,
			Amount:            req.Amount,
			Currency:          lockedReceiver.Currency,
			Type:              "DEPOSIT",
			Status:            "SUCCESS",
			Description:       formattedDesc,
		}

		if err := s.repo.CreateTransaction(tx, newTransaction); err != nil {
			return err
		}
		if err := CreateDoubleEntry(
			tx,
			newTransaction.ID,
			lockedAdmin.ID,
			lockedReceiver.ID,
			req.Amount,
			lockedReceiver.Currency,
			adminNewBalance,
			receiverNewBalance,
		); err != nil {
			return err
		}

		// Tạo thông báo biến động số dư cho người nhận (nạp tiền)
		receiverMsg := fmt.Sprintf("Tài khoản của bạn đã được cộng +%d VND từ giao dịch nạp tiền Admin. Số dư mới: %d VND. Nội dung: %s", req.Amount, receiverNewBalance, formattedDesc)
		if err := s.notiService.CreateNotification(tx, lockedReceiver.UserID, "BALANCE_FLUCTUATION", "Biến động số dư (+)", receiverMsg); err != nil {
			return err
		}

		transactionResult = newTransaction
		receiverPushMessage = receiverMsg
		return nil
	})

	if err != nil {
		existing, findErr := s.repo.FindTransactionByIdempotencyKey(
			adminUserID,
			req.IdempotencyKey,
		)
		if findErr == nil && existing != nil {
			return mapTransactionResponse(existing), nil
		}
		return nil, err
	}

	_ = s.notiService.SendPushToUser(
		receiverAccount.UserID,
		"Biến động số dư (+)",
		receiverPushMessage,
		map[string]string{
			"type":           "BALANCE_FLUCTUATION",
			"reference_code": transactionResult.ReferenceCode,
			"transaction":    "DEPOSIT",
		},
	)
	return mapTransactionResponse(transactionResult), nil
}

func (s *Service) GetTransactionsByAccountID(accountID uint) ([]TransactionResponse, error) {
	transactions, err := s.repo.FindTransactionViewsByAccountID(accountID)
	if err != nil {
		return nil, err
	}

	response := make([]TransactionResponse, 0)
	for _, transaction := range transactions {
		response = append(response, mapTransactionView(transaction))
	}

	return response, nil
}

func mapTransactionView(transaction transactionView) TransactionResponse {
	return TransactionResponse{
		ID:                        transaction.ID,
		ReferenceCode:             transaction.ReferenceCode,
		SenderAccountID:           transaction.SenderAccountID,
		ReceiverAccountID:         transaction.ReceiverAccountID,
		Amount:                    transaction.Amount,
		Currency:                  transaction.Currency,
		Type:                      transaction.Type,
		Status:                    transaction.Status,
		Description:               transaction.Description,
		Direction:                 transaction.Direction,
		CounterpartyName:          transaction.CounterpartyName,
		CounterpartyAccountNumber: transaction.CounterpartyAccountNumber,
		BalanceAfter:              transaction.BalanceAfter,
		CreatedAt:                 transaction.CreatedAt,
	}
}
