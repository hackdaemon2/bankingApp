package repository

import (
	"bankingApp/internal/model"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

// NewAccountRepository creates a new instance of AccountRepository
func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{
		db: db,
	}
}

// UpdateAccount updates an account in the database within a transaction
func (a AccountRepository) UpdateAccount(account *model.Account) error {
	tx := a.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&model.Account{}).Where(model.Account{AccountID: account.AccountID}).
		UpdateColumns(map[string]interface{}{
			"balance":    account.Balance,
			"created_at": time.Now(),
		}).Error; err != nil {
		slog.Error(err.Error())
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		tx.Rollback()
		return err
	}

	return nil
}

// GetAccountByAccountNumber fetch user account details by account number
func (a AccountRepository) GetAccountByAccountNumber(number string) (*model.Account, error) {
	var account model.Account
	err := a.db.
		Where(&model.Account{AccountNumber: number}).
		First(&account).
		Error
	return &account, err
}
