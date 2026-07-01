package admin

import "time"

type AdminUserResponse struct {
	ID         uint      `json:"id"`
	FullName   string    `json:"full_name"`
	Phone      string    `json:"phone"`
	Role       string    `json:"role"`
	IsVerified bool      `json:"is_verified"`
	IsLocked   bool      `json:"is_locked"`
	AvatarURL  string    `json:"avatar_url"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateAdminRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

type StepUpRequest struct {
	Action   string `json:"action" binding:"required"`
	TOTPCode string `json:"totp_code" binding:"required,len=6"`
	Binding  string `json:"binding" binding:"required,max=500"`
}

type StepUpResponse struct {
	Token     string    `json:"token"`
	Action    string    `json:"action"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CreateAdminResponse struct {
	ID         uint      `json:"id"`
	FullName   string    `json:"full_name"`
	Phone      string    `json:"phone"`
	Role       string    `json:"role"`
	TOTPSecret string    `json:"totp_secret"`
	CreatedAt  time.Time `json:"created_at"`
}

type AdminTransactionSummary struct {
	ReferenceCode string    `json:"reference_code"`
	Type          string    `json:"type"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

type AuditLogResponse struct {
	ID          uint      `json:"id"`
	ActorUserID uint      `json:"actor_user_id"`
	ActorName   string    `json:"actor_name"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	Summary     string    `json:"summary"`
	IPAddress   string    `json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`
}

type DashboardResponse struct {
	CustomerCount         int64                     `json:"customer_count"`
	AdminCount            int64                     `json:"admin_count"`
	LockedCustomerCount   int64                     `json:"locked_customer_count"`
	PaymentBalance        int64                     `json:"payment_balance"`
	ActiveSavingsCount    int64                     `json:"active_savings_count"`
	ActiveSavingsBalance  int64                     `json:"active_savings_balance"`
	TodayTransactionCount int64                     `json:"today_transaction_count"`
	TodayTransactionValue int64                     `json:"today_transaction_value"`
	RecentUsers           []AdminUserResponse       `json:"recent_users"`
	RecentTransactions    []AdminTransactionSummary `json:"recent_transactions"`
	RecentAuditLogs       []AuditLogResponse        `json:"recent_audit_logs"`
}
