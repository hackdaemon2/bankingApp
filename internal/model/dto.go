package model

import (
	"encoding/json"
	"fmt"

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

type Money struct {
	decimal.Decimal
}

func (d Money) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil // Omit if zero-valued
	}
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

func (d Money) IsZero() bool {
	return d.Decimal.IsZero()
}

func (d Money) String() string {
	return d.Decimal.String()
}

type ThirdPartyTransactionDataDTO struct {
	AccountID string `json:"account_id,omitempty" mapstructure:"account_id"`
	Reference string `json:"reference,omitempty"`
	Amount    Money  `json:"amount,omitempty"`
}

func (dto ThirdPartyTransactionDataDTO) MarshalJSON() ([]byte, error) {
	type Alias ThirdPartyTransactionDataDTO
	alias := &struct {
		*Alias
		Amount string `json:"amount,omitempty"`
	}{
		Alias: (*Alias)(&dto),
	}

	if dto.Amount.IsZero() {
		alias.Amount = ""
	} else {
		alias.Amount = dto.Amount.String()
	}

	return json.Marshal(alias)
}

type DebitRequestDTO struct {
	ThirdPartyTransactionDataDTO
}

type CreditRequestDTO struct {
	ThirdPartyTransactionDataDTO
}

type ResponseDTO struct {
	ThirdPartyTransactionDataDTO
	PaymentReference string `json:"payment_reference,omitempty"`
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
	Amount         Money           `json:"amount" validate:"required,isPositive"`
	Type           TransactionType `json:"type" validate:"required,oneof=credit debit"`
}

type TransactionRequestDTO struct {
	TransactionDataDTO
}

type TransactionResponseDTO struct {
	TransactionDataDTO
	Success bool `json:"success"`
}
