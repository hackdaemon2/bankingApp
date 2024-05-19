package model

import (
	"sync"
	"time"

	"github.com/govalues/decimal"
)

type TimestampData struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	UserID         uint   `gorm:"primaryKey"`
	Username       string `gorm:"index:idx_username;unique"`
	Password       string
	TransactionPin string
	Accounts       []Account
	TimestampData
}

type Account struct {
	AccountID     uint   `gorm:"primaryKey"`
	UserID        uint   // Foreign key referencing the User table
	AccountNumber string `gorm:"index:idx_account_number;unique"`
	Balance       Money
	mu            sync.Mutex `gorm:"-"`
	TimestampData
}

func (acc *Account) SetBalance(value Money) {
	acc.Balance = value
}

func (acc *Account) GetBalance() Money {
	return acc.Balance
}

const scale = 2

func (acc *Account) Deposit(amount Money) error {
	acc.mu.Lock()
	defer acc.mu.Unlock()
	floatVal, _ := amount.Float64()
	val, _ := decimal.NewFromFloat64(floatVal)
	newBalance, err := acc.GetBalance().AddExact(val, scale)
	if err != nil {
		return err
	}
	acc.SetBalance(Money{Decimal: newBalance})
	return nil
}

func (acc *Account) Withdraw(amount Money) error {
	acc.mu.Lock()
	defer acc.mu.Unlock()
	floatVal, _ := amount.Float64()
	val, _ := decimal.NewFromFloat64(floatVal)
	newBalance, err := acc.GetBalance().SubExact(val, scale)
	if err != nil {
		return err
	}
	acc.SetBalance(Money{Decimal: newBalance})
	return nil
}

const insufficientBalanceFlag = -1

func (acc *Account) IsInsufficientBalance(amount Money) bool {
	acc.mu.Lock()
	defer acc.mu.Unlock()
	floatVal, _ := amount.Float64()
	val, _ := decimal.NewFromFloat64(floatVal)
	return acc.GetBalance().Cmp(val) == insufficientBalanceFlag
}

type Transaction struct {
	TransactionID    uint   `gorm:"primaryKey"`
	AccountID        uint   `gorm:"index"`
	Reference        string `gorm:"index:idx_reference;unique"`
	PaymentReference string `gorm:"column:payment_reference;index:idx_payment_reference;unique"`
	Amount           Money
	Type             TransactionType
	Success          bool
	TransactionTime  time.Time
	TimestampData
}
