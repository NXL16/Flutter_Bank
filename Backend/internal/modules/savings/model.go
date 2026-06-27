package savings

import (
	"bank-service/internal/modules/account"
	"time"
)

type SavingsDetail struct {
	ID                uint            `gorm:"primaryKey" json:"id"`
	AccountID         uint            `gorm:"not null;uniqueIndex" json:"account_id"`
	Account           account.Account `gorm:"foreignKey:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"account,omitempty"`
	InterestRate      float64         `gorm:"not null;type:decimal(5,2);default:5.50" json:"interest_rate"`
	TermMonths        int             `gorm:"not null;default:12" json:"term_months"`
	StartDate         time.Time       `gorm:"not null" json:"start_date"`
	EndDate           time.Time       `gorm:"not null" json:"end_date"`
	OriginalPrincipal int64           `gorm:"not null" json:"original_principal"`
	IsSettled         bool            `gorm:"default:false" json:"is_settled"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}
