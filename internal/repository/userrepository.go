package repository

import (
	"bankingApp/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct { // define UserRepository
	DB *gorm.DB
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// FindUserByUsername retrieves a user by username from the database
func (u *UserRepository) FindUserByUsername(username string) (model.User, error) {
	var user model.User
	u.DB.
		Where(&model.User{Username: username}).
		Find(&user)
	return user, nil
}

// GetUserByAccountNumber retrieves a user by account number from the database
func (u *UserRepository) GetUserByAccountNumber(accountNumber string) (*model.User, *model.Account, error) {
	var user model.User
	var account model.Account
	userResult := u.DB.Joins("JOIN tbl_account ON tbl_user.user_id = tbl_account.user_id").
		Where("tbl_account.account_number = ?", accountNumber).
		First(&user)
	if userResult.Error != nil {
		return nil, nil, userResult.Error
	}

	// If the user is found, also fetch the associated account
	if user.UserID != 0 {
		accountResult := u.DB.Where("user_id = ?", user.UserID).First(&account)
		if accountResult.Error != nil {
			return nil, nil, accountResult.Error
		}
	}

	return &user, &account, nil
}
