package model //nolint:typecheck

import (
	"bankingApp/internal/api/constants"
	"encoding/json"
	"errors"
	"strconv"

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
	AccountID string      `json:"account_id,omitempty" mapstructure:"account_id"`
	Reference string      `json:"reference,omitempty"`
	Amount    *BigDecimal `json:"amount,omitempty"`
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

type BigDecimal struct {
	decimal.Decimal
}

func (a *BigDecimal) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d, _ := decimal.NewFromFloat64(value)
		a.Decimal = d
	case string:
		const bitSize = 64
		numericValue, err := strconv.ParseFloat(value, bitSize)
		if err != nil {
			a.Decimal = decimal.NegOne
			return nil
		}
		d, err := decimal.NewFromFloat64(numericValue)
		if err != nil {
			a.Decimal = decimal.NegOne
			return err
		}
		a.Decimal = d
	case nil:
		a.Decimal = decimal.NegOne
	default:
		a.Decimal = decimal.NegOne
		return errors.New("invalid type for amount")
	}
	return nil
}

const coeff = -1

func (a BigDecimal) MarshalJSON() ([]byte, error) {
	if a.Decimal.Cmp(decimal.MustNew(coeff, constants.Zero)) == constants.Zero {
		return nil, nil
	}
	return json.Marshal(a.Decimal)
}

type TransactionDataDTO struct {
	AccountNumber  string          `json:"account_number" validate:"required,min=10,max=10"`
	Username       string          `json:"username" validate:"required"`
	TransactionPin string          `json:"transaction_pin" validate:"required,min=4,max=4"`
	Reference      string          `json:"payment_reference" validate:"required,min=1,max=255"`
	Amount         BigDecimal      `json:"amount" validate:"required,isPositive"`
	Type           TransactionType `json:"type" validate:"required,oneof=credit debit"`
}

type TransactionRequestDTO struct {
	TransactionDataDTO
}

type TransactionResponseDTO struct {
	TransactionDataDTO
	Success bool `json:"success"`
}
