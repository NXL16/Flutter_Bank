package admin

import (
	"bank-service/internal/infrastructure/totp"
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/auth"
	"bank-service/internal/modules/notification"
	"bank-service/internal/modules/transaction"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	repo                *Repository
	accountService      *account.Service
	transactionService  *transaction.Service
	notificationService *notification.Service
}

func NewService(
	repo *Repository,
	accountService *account.Service,
	transactionService *transaction.Service,
	notificationService *notification.Service,
) *Service {
	return &Service{
		repo:                repo,
		accountService:      accountService,
		transactionService:  transactionService,
		notificationService: notificationService,
	}
}

const (
	ActionDeposit     = "DEPOSIT"
	ActionCreateAdmin = "CREATE_ADMIN"
	ActionLockUser    = "LOCK_USER"
	ActionUnlockUser  = "UNLOCK_USER"
)

var sensitiveAdminActions = map[string]struct{}{
	ActionDeposit:     {},
	ActionCreateAdmin: {},
	ActionLockUser:    {},
	ActionUnlockUser:  {},
}

func (s *Service) CreateStepUp(
	actorUserID uint,
	req StepUpRequest,
) (*StepUpResponse, error) {
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	binding := strings.TrimSpace(req.Binding)
	if _, allowed := sensitiveAdminActions[action]; !allowed {
		return nil, errors.New("Hành động xác thực nâng cao không hợp lệ")
	}
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(req.TOTPCode) {
		return nil, errors.New("Mã TOTP phải gồm đúng 6 số")
	}
	if binding == "" || len(binding) > 500 {
		return nil, errors.New("Dữ liệu ràng buộc xác thực không hợp lệ")
	}
	adminUser, err := s.repo.FindUserByID(actorUserID)
	if err != nil {
		return nil, err
	}
	if adminUser.IsLocked ||
		(adminUser.Role != "admin" && adminUser.Role != "super_admin") {
		return nil, errors.New("Tài khoản không có quyền quản trị")
	}

	now := time.Now()
	if adminUser.TOTPLockedUntil != nil &&
		adminUser.TOTPLockedUntil.After(now) {
		return nil, fmt.Errorf(
			"Xác thực nâng cao đang bị khóa, thử lại sau %d phút",
			int(time.Until(*adminUser.TOTPLockedUntil).Minutes())+1,
		)
	}

	timeStep, valid := totp.MatchCodeAt(
		adminUser.TOTPSecret,
		req.TOTPCode,
		now,
	)
	if !valid {
		lockedUntil, recordErr := s.repo.RecordMFAFailure(
			actorUserID,
			now,
			5,
			10*time.Minute,
		)
		if recordErr != nil {
			return nil, recordErr
		}
		if lockedUntil != nil {
			s.notifyAdmin(
				actorUserID,
				"Cảnh báo bảo mật Admin",
				"Xác thực nâng cao đã bị khóa 10 phút sau nhiều lần nhập sai TOTP.",
				"ADMIN_SECURITY",
			)
			return nil, errors.New(
				"Nhập sai TOTP quá nhiều lần, xác thực bị khóa 10 phút",
			)
		}
		return nil, errors.New("Mã TOTP không đúng hoặc đã hết hạn")
	}

	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(rawToken)
	tokenHash := sha256.Sum256([]byte(token))
	bindingHash := sha256.Sum256([]byte(binding))
	expiresAt := now.Add(2 * time.Minute)
	challenge := &StepUpChallenge{
		ActorUserID:  actorUserID,
		Action:       action,
		TokenHash:    hex.EncodeToString(tokenHash[:]),
		BindingHash:  hex.EncodeToString(bindingHash[:]),
		TOTPTimeStep: timeStep,
		ExpiresAt:    expiresAt,
	}
	if err := s.repo.CreateStepUpChallenge(challenge); err != nil {
		if errors.Is(err, errTOTPAlreadyUsed) {
			return nil, errors.New(
				"Mã TOTP này đã được sử dụng, vui lòng chờ mã mới",
			)
		}
		used, lookupErr := s.repo.HasTOTPUsage(actorUserID, timeStep)
		if lookupErr == nil && used {
			return nil, errors.New(
				"Mã TOTP này đã được sử dụng, vui lòng chờ mã mới",
			)
		}
		return nil, err
	}
	return &StepUpResponse{
		Token:     token,
		Action:    action,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) AuthorizeStepUp(
	actorUserID uint,
	action string,
	token string,
	binding string,
) error {
	action = strings.ToUpper(strings.TrimSpace(action))
	token = strings.TrimSpace(token)
	binding = strings.TrimSpace(binding)
	if _, allowed := sensitiveAdminActions[action]; !allowed {
		return errors.New("Hành động xác thực nâng cao không hợp lệ")
	}
	if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(token) {
		return errors.New("Thiếu xác thực nâng cao cho thao tác này")
	}
	tokenHash := sha256.Sum256([]byte(token))
	bindingHash := sha256.Sum256([]byte(binding))
	consumed, err := s.repo.ConsumeStepUpChallenge(
		actorUserID,
		action,
		hex.EncodeToString(tokenHash[:]),
		hex.EncodeToString(bindingHash[:]),
		time.Now(),
	)
	if err != nil {
		return err
	}
	if !consumed {
		return errors.New("Xác thực nâng cao đã hết hạn hoặc đã được sử dụng")
	}
	return nil
}

func userActionBinding(action string, userID uint) string {
	return fmt.Sprintf("%s|%d", action, userID)
}

func createAdminBinding(req CreateAdminRequest) string {
	return fmt.Sprintf(
		"%s|%s|%s",
		ActionCreateAdmin,
		strings.TrimSpace(req.FullName),
		strings.TrimSpace(req.Phone),
	)
}

func depositBinding(req transaction.DepositRequest) string {
	return fmt.Sprintf(
		"%s|%s|%d|%s",
		ActionDeposit,
		strings.TrimSpace(req.ReceiverAccountNumber),
		req.Amount,
		strings.TrimSpace(req.Description),
	)
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

func (s *Service) GetDashboard() (*DashboardResponse, error) {
	now := time.Now()
	startOfDay := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		now.Location(),
	)
	metrics, err := s.repo.DashboardMetrics(startOfDay)
	if err != nil {
		return nil, err
	}
	recentUsers, err := s.repo.FindRecentUsers(6)
	if err != nil {
		return nil, err
	}
	recentTransactions, err := s.repo.FindRecentTransactions(6)
	if err != nil {
		return nil, err
	}
	auditLogs, err := s.repo.FindRecentAuditLogs(6)
	if err != nil {
		return nil, err
	}
	users := make([]AdminUserResponse, 0, len(recentUsers))
	for _, item := range recentUsers {
		users = append(users, mapUserToAdminResponse(item))
	}
	return &DashboardResponse{
		CustomerCount:         metrics.CustomerCount,
		AdminCount:            metrics.AdminCount,
		LockedCustomerCount:   metrics.LockedCustomerCount,
		PaymentBalance:        metrics.PaymentBalance,
		ActiveSavingsCount:    metrics.ActiveSavingsCount,
		ActiveSavingsBalance:  metrics.ActiveSavingsBalance,
		TodayTransactionCount: metrics.TodayTransactionCount,
		TodayTransactionValue: metrics.TodayTransactionValue,
		RecentUsers:           users,
		RecentTransactions:    recentTransactions,
		RecentAuditLogs:       auditLogs,
	}, nil
}

func (s *Service) GetTransactions(
	limit int,
) ([]AdminTransactionSummary, error) {
	return s.repo.FindRecentTransactions(limit)
}

func (s *Service) GetAuditLogs(limit int) ([]AuditLogResponse, error) {
	return s.repo.FindRecentAuditLogs(limit)
}

func (s *Service) GetUserByID(userID uint) (*AdminUserResponse, error) {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return nil, err
	}

	resp := mapUserToAdminResponse(*user)
	return &resp, nil
}

func (s *Service) LockUser(
	actorUserID uint,
	userID uint,
	ipAddress string,
) error {
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

	if err := s.repo.LockUser(
		actorUserID,
		userID,
		ipAddress,
		fmt.Sprintf("Khóa tài khoản %s (%s)", user.FullName, user.Phone),
	); err != nil {
		return err
	}
	s.notifyAdmin(
		actorUserID,
		"Đã khóa tài khoản",
		fmt.Sprintf("Bạn đã khóa tài khoản %s (%s).", user.FullName, user.Phone),
		"ADMIN_OPERATION",
	)
	s.notifyAdmin(
		userID,
		"Tài khoản đã bị khóa",
		"Tài khoản của bạn đã bị khóa bởi bộ phận quản trị. Vui lòng liên hệ hỗ trợ nếu cần.",
		"ACCOUNT_SECURITY",
	)
	return nil
}

func (s *Service) UnlockUser(
	actorUserID uint,
	userID uint,
	ipAddress string,
) error {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}

	if user.Role != "user" {
		return errors.New("Không thể mở khóa tài khoản quản trị viên")
	}

	if !user.IsLocked {
		return errors.New("Tài khoản chưa bị khóa")
	}

	if err := s.repo.UnlockUser(
		actorUserID,
		userID,
		ipAddress,
		fmt.Sprintf("Mở khóa tài khoản %s (%s)", user.FullName, user.Phone),
	); err != nil {
		return err
	}
	s.notifyAdmin(
		actorUserID,
		"Đã mở khóa tài khoản",
		fmt.Sprintf(
			"Bạn đã mở khóa tài khoản %s (%s).",
			user.FullName,
			user.Phone,
		),
		"ADMIN_OPERATION",
	)
	s.notifyAdmin(
		userID,
		"Tài khoản đã được mở khóa",
		"Bạn có thể đăng nhập và sử dụng NF Bank trở lại.",
		"ACCOUNT_SECURITY",
	)
	return nil
}

func mapUserToAdminResponse(user auth.User) AdminUserResponse {
	return AdminUserResponse{
		ID:         user.ID,
		FullName:   user.FullName,
		Phone:      user.Phone,
		Role:       user.Role,
		IsVerified: user.IsVerified,
		IsLocked:   user.IsLocked,
		AvatarURL:  user.Profile.AvatarURL,
		CreatedAt:  user.CreatedAt,
	}
}

func (s *Service) GetUserAccounts(
	userID uint,
) ([]account.AccountResponse, error) {
	if _, err := s.repo.FindUserByID(userID); err != nil {
		return nil, err
	}
	return s.accountService.GetUserAccounts(userID)
}

func (s *Service) CreateAdmin(
	actorUserID uint,
	ipAddress string,
	req CreateAdminRequest,
) (*CreateAdminResponse, error) {
	req.FullName = strings.TrimSpace(req.FullName)
	if len([]rune(req.FullName)) < 2 || len([]rune(req.FullName)) > 100 {
		return nil, errors.New("Họ và tên phải có từ 2 đến 100 ký tự")
	}
	if err := validateAdminPassword(req.Password); err != nil {
		return nil, err
	}
	phone := "+84" + normalizePhone(req.Phone)
	if !regexp.MustCompile(`^\+84(3|5|7|8|9)[0-9]{8}$`).MatchString(phone) {
		return nil, errors.New("Số điện thoại Việt Nam không hợp lệ")
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

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		txRepo := s.repo.withDB(tx)
		existingUser, findErr := txRepo.FindUserByPhone(phone)
		if findErr != nil {
			return findErr
		}
		if existingUser != nil {
			return errors.New("Số điện thoại đã được sử dụng")
		}
		if createErr := txRepo.CreateAdminUser(adminUser); createErr != nil {
			return createErr
		}
		return txRepo.CreateAuditLog(&AuditLog{
			ActorUserID: actorUserID,
			Action:      "CREATE_ADMIN",
			TargetType:  "USER",
			TargetID:    fmt.Sprintf("%d", adminUser.ID),
			Summary: fmt.Sprintf(
				"Tạo quản trị viên %s (%s)",
				adminUser.FullName,
				adminUser.Phone,
			),
			IPAddress: ipAddress,
		})
	})
	if err != nil {
		return nil, err
	}
	s.notifyAdmin(
		actorUserID,
		"Đã tạo quản trị viên",
		fmt.Sprintf(
			"Bạn đã tạo tài khoản Admin %s (%s).",
			adminUser.FullName,
			adminUser.Phone,
		),
		"ADMIN_SECURITY",
	)

	return &CreateAdminResponse{
		ID:         adminUser.ID,
		FullName:   adminUser.FullName,
		Phone:      adminUser.Phone,
		Role:       adminUser.Role,
		TOTPSecret: adminUser.TOTPSecret,
		CreatedAt:  adminUser.CreatedAt,
	}, nil
}

func validateAdminPassword(password string) error {
	if len(password) < 12 {
		return errors.New("Mật khẩu Admin phải có ít nhất 12 ký tự")
	}
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) ||
		!regexp.MustCompile(`[a-z]`).MatchString(password) ||
		!regexp.MustCompile(`[0-9]`).MatchString(password) ||
		!regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password) {
		return errors.New(
			"Mật khẩu Admin phải có chữ hoa, chữ thường, số và ký tự đặc biệt",
		)
	}
	return nil
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
	result, err := s.transactionService.Deposit(adminUserID, req)
	if err != nil {
		return nil, err
	}
	s.notifyAdmin(
		adminUserID,
		"Nạp tiền đã hoàn tất",
		fmt.Sprintf(
			"Giao dịch %s đã cấp %d VND cho tài khoản %s.",
			result.ReferenceCode,
			result.Amount,
			req.ReceiverAccountNumber,
		),
		"ADMIN_OPERATION",
	)
	return result, nil
}

func (s *Service) GetAccountTransactions(accountID uint) ([]transaction.TransactionResponse, error) {
	return s.transactionService.GetTransactionsByAccountID(accountID)
}

func (s *Service) notifyAdmin(
	userID uint,
	title string,
	content string,
	notificationType string,
) {
	if s.notificationService == nil {
		return
	}
	_ = s.notificationService.CreateUserNotification(
		userID,
		notificationType,
		title,
		content,
	)
	_ = s.notificationService.SendPushToUser(
		userID,
		title,
		content,
		map[string]string{"type": notificationType},
	)
}
