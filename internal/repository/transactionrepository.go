package repository

import (
	"bankingApp/internal/model"

	"gorm.io/gorm"
)

type TransactionRepository struct { // TransactionRepository definition
	db *gorm.DB
}

// NewTransactionRepository creates a new instance of TransactionRepository with the provided gorm.DB instance.
func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// FindTransaction retrieves a transaction by ID from the database
func (t *TransactionRepository) FindTransaction(id uint) (*model.Transaction, error) {
	var transaction model.Transaction
	err := t.db.
		Where(&model.Transaction{TransactionID: id}).
		Find(&transaction).
		Error
	return &transaction, err
}

// FindTransactionByReference validates that a transaction exists using the unique reference from the database
func (t *TransactionRepository) FindTransactionByReference(reference string) (*model.Transaction, error) {
	var transaction model.Transaction
	err := t.db.
		Where(&model.Transaction{PaymentReference: reference}).
		Find(&transaction).
		Error
	return &transaction, err
}

// GetLastInsertID returns the last inserted transaction ID from the database.
func (t *TransactionRepository) GetLastInsertID() (uint, error) {
	var transaction model.Transaction
	err := t.db.Order("transaction_id DESC").Limit(1).Find(&transaction).Error
	if err != nil {
		return 0, err
	}
	return transaction.TransactionID, nil
}

// SaveTransaction saves the transaction details to the DB
func (t *TransactionRepository) SaveTransaction(transaction *model.Transaction) error {
	tx := t.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	return nil
}
