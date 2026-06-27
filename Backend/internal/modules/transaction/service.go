package transaction

import (
	"bank-service/internal/infrastructure/firebase"
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/notification"
	"errors"
	"fmt"
	"regexp"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	repo           *Repository
	firebaseClient *firebase.Client
	notiService    *notification.Service
}

func NewService(repo *Repository, firebaseClient *firebase.Client, notiService *notification.Service) *Service {
	return &Service{
		repo:           repo,
		firebaseClient: firebaseClient,
		notiService:    notiService,
	}
}

func (s *Service) Transfer(
	userID uint,
	req TransferRequest,
) (*TransactionResponse, error) {

	if req.IDToken == "" {
		return nil, errors.New("giao dịch chuyển tiền yêu cầu xác thực OTP")
	}

	verifiedPhone, err := s.firebaseClient.VerifyIDToken(req.IDToken)
	if err != nil {
		return nil, err
	}

	userPhone, err := s.repo.GetUserPhone(userID)
	if err != nil {
		return nil, errors.New("không thể xác thực thông tin số điện thoại của người dùng")
	}

	if normalizePhone(verifiedPhone) != normalizePhone(userPhone) {
		return nil, errors.New("số điện thoại xác thực OTP không trùng khớp với số điện thoại đăng ký tài khoản")
	}

	var transactionResult *Transaction

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		// 1. Tìm sender account (chưa lock) để lấy ID
		senderAccount, err := s.repo.FindPaymentAccountByUserID(userID)
		if err != nil {
			return err
		}
		if senderAccount == nil {
			return errors.New("không tìm thấy tài khoản gửi")
		}

		// 2. Tìm receiver account (chưa lock) để lấy ID
		receiverAccount, err := s.repo.FindAccountByNumber(req.ReceiverAccountNumber)
		if err != nil {
			return err
		}
		if receiverAccount == nil {
			return errors.New("không tìm thấy tài khoản nhận")
		}

		if senderAccount.ID == receiverAccount.ID {
			return errors.New("không thể chuyển tiền cho chính tài khoản của mình")
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
			return errors.New("không tìm thấy tài khoản gửi")
		}
		if lockedReceiver == nil {
			return errors.New("không tìm thấy tài khoản nhận")
		}

		if lockedSender.Status != "ACTIVE" {
			return errors.New("tài khoản gửi không hoạt động")
		}

		if lockedReceiver.Status != "ACTIVE" {
			return errors.New("tài khoản nhận không hoạt động")
		}

		if lockedSender.Currency != lockedReceiver.Currency {
			return errors.New("không thể chuyển tiền khác loại tiền tệ")
		}

		if lockedSender.Balance < req.Amount {
			return errors.New("số dư không đủ")
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
		newTransaction := &Transaction{
			ReferenceCode:     generateReferenceCode(),
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

		return nil
	})

	if err != nil {
		return nil, err
	}

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
	}, nil
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
		return nil, errors.New("không tìm thấy tài khoản PAYMENT")
	}

	transactions, err := s.repo.FindTransactionsByAccountID(paymentAccount.ID)
	if err != nil {
		return nil, err
	}

	response := make([]TransactionResponse, 0)

	for _, transaction := range transactions {
		response = append(response, TransactionResponse{
			ID:                transaction.ID,
			ReferenceCode:     transaction.ReferenceCode,
			SenderAccountID:   transaction.SenderAccountID,
			ReceiverAccountID: transaction.ReceiverAccountID,
			Amount:            transaction.Amount,
			Currency:          transaction.Currency,
			Type:              transaction.Type,
			Status:            transaction.Status,
			Description:       transaction.Description,
		})
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
		return nil, errors.New("không tìm thấy tài khoản PAYMENT")
	}

	transaction, err := s.repo.FindTransactionByReferenceCode(referenceCode)
	if err != nil {
		return nil, err
	}

	if transaction == nil {
		return nil, errors.New("không tìm thấy giao dịch")
	}

	isOwner :=
		(transaction.SenderAccountID != nil && *transaction.SenderAccountID == paymentAccount.ID) ||
			transaction.ReceiverAccountID == paymentAccount.ID

	if !isOwner {
		return nil, errors.New("không có quyền truy cập giao dịch này")
	}

	return &TransactionResponse{
		ID:                transaction.ID,
		ReferenceCode:     transaction.ReferenceCode,
		SenderAccountID:   transaction.SenderAccountID,
		ReceiverAccountID: transaction.ReceiverAccountID,
		Amount:            transaction.Amount,
		Currency:          transaction.Currency,
		Type:              transaction.Type,
		Status:            transaction.Status,
		Description:       transaction.Description,
	}, nil
}

func normalizePhone(phone string) string {
	reg := regexp.MustCompile(`\D`)
	digits := reg.ReplaceAllString(phone, "")

	if len(digits) >= 11 && digits[:2] == "84" {
		return digits[2:]
	}
	if len(digits) > 0 && digits[:1] == "0" {
		return digits[1:]
	}
	return digits
}

func (s *Service) Deposit(adminUserID uint, req DepositRequest) (*TransactionResponse, error) {
	// 1. Tìm tài khoản nguồn (PAYMENT) của Admin
	adminPaymentAccount, err := s.repo.FindPaymentAccountByUserID(adminUserID)
	if err != nil {
		return nil, err
	}
	if adminPaymentAccount == nil {
		return nil, errors.New("không tìm thấy tài khoản nguồn PAYMENT của admin (vui lòng liên hệ hỗ trợ)")
	}

	// 2. Tìm tài khoản nhận (PAYMENT) của User bằng số tài khoản
	receiverAccount, err := s.repo.FindAccountByNumber(req.ReceiverAccountNumber)
	if err != nil {
		return nil, err
	}
	if receiverAccount == nil {
		return nil, errors.New("không tìm thấy số tài khoản người nhận")
	}

	if adminPaymentAccount.ID == receiverAccount.ID {
		return nil, errors.New("không thể tự nạp tiền cho chính tài khoản Admin của mình")
	}

	// 3. Tìm thông tin tên Admin để ghi nhận lịch sử kiểm toán (audit log)
	var adminUser struct {
		FullName string
	}
	err = s.repo.db.Table("users").Where("id = ?", adminUserID).First(&adminUser).Error
	if err != nil {
		return nil, errors.New("không thể tìm thấy thông tin định danh của admin thực hiện")
	}

	var transactionResult *Transaction

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		// Khóa 2 tài khoản theo thứ tự ID để chống deadlock
		var lockedAdmin, lockedReceiver *account.Account
		if adminPaymentAccount.ID < receiverAccount.ID {
			lockedAdmin, err = s.repo.FindAccountByIDForUpdate(tx, adminPaymentAccount.ID)
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
			lockedAdmin, err = s.repo.FindAccountByIDForUpdate(tx, adminPaymentAccount.ID)
			if err != nil {
				return err
			}
		}

		if lockedAdmin == nil || lockedReceiver == nil {
			return errors.New("không tìm thấy thông tin tài khoản")
		}

		if lockedReceiver.Status != "ACTIVE" {
			return errors.New("tài khoản người nhận đang bị khóa hoặc không hoạt động")
		}

		if lockedAdmin.Currency != lockedReceiver.Currency {
			return errors.New("không thể chuyển tiền khác loại tiền tệ")
		}

		// Nạp tiền: Ví Admin giảm (cho phép âm), ví User tăng
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

		// Tạo thông báo biến động số dư cho người nhận (nạp tiền)
		receiverMsg := fmt.Sprintf("Tài khoản của bạn đã được cộng +%d VND từ giao dịch nạp tiền Admin. Số dư mới: %d VND. Nội dung: %s", req.Amount, receiverNewBalance, formattedDesc)
		if err := s.notiService.CreateNotification(tx, lockedReceiver.UserID, "BALANCE_FLUCTUATION", "Biến động số dư (+)", receiverMsg); err != nil {
			return err
		}

		transactionResult = newTransaction
		return nil
	})

	if err != nil {
		return nil, err
	}

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
	}, nil
}

func (s *Service) GetTransactionsByAccountID(accountID uint) ([]TransactionResponse, error) {
	transactions, err := s.repo.FindTransactionsByAccountID(accountID)
	if err != nil {
		return nil, err
	}

	response := make([]TransactionResponse, 0)
	for _, transaction := range transactions {
		response = append(response, TransactionResponse{
			ID:                transaction.ID,
			ReferenceCode:     transaction.ReferenceCode,
			SenderAccountID:   transaction.SenderAccountID,
			ReceiverAccountID: transaction.ReceiverAccountID,
			Amount:            transaction.Amount,
			Currency:          transaction.Currency,
			Type:              transaction.Type,
			Status:            transaction.Status,
			Description:       transaction.Description,
		})
	}

	return response, nil
}

