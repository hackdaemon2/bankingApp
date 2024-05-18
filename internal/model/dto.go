package model

import (
	"time"

	"github.com/govalues/decimal"
)

type IAppConfiguration interface {
	ReadTimeout() uint32
	ServerPort() uint32
	ThirdPartyBaseUrl() string
	GinMode() string
	Username() string
	Password() string
	Host() string
	Port() int
	DatabaseName() string
	MaximumOpenConnection() int
	MaximumIdleConnection() int
	MaximumIdleTime() int
	MaximumTime() int
	JwtSecret() string
}

type ThirdPartyTransactionDataDTO struct {
	AccountID string          `json:"account_id"`
	Reference string          `json:"reference"`
	Amount    decimal.Decimal `json:"amount"`
}

type DebitRequestDTO struct {
	ThirdPartyTransactionDataDTO
}

type CreditRequestDTO struct {
	ThirdPartyTransactionDataDTO
}

type ResponseDTO struct {
	ThirdPartyTransactionDataDTO
	PaymentReference string `json:"payment_reference"`
}

type AuthenticationResponseDTO struct {
	CreatedAt   time.Time `json:"created_at,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	AccessToken string    `json:"access_token,omitempty"`
	Success     bool      `json:"success"`
	Message     string    `json:"message"`
}

type TransactionType string

const (
	DebitTransaction  TransactionType = "debit"
	CreditTransaction TransactionType = "credit"
)

type TransactionDataDTO struct {
	AccountNumber  string          `json:"account_number" validate:"required,min=10,max=10"`
	Username       string          `json:"username" validate:"required"`
	TransactionPin string          `json:"transaction_pin" validate:"required,min=4,max=4"`
	Reference      string          `json:"payment_reference" validate:"required,min=1,max=255"`
	Amount         decimal.Decimal `json:"amount" validate:"required,isPositive"`
	Type           TransactionType `json:"type" validate:"required,oneof=credit debit"`
}

type TransactionRequestDTO struct {
	TransactionDataDTO
}

type TransactionResponseDTO struct {
	TransactionDataDTO
	Success bool `json:"success"`
}
