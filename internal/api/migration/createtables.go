package migration

import (
	"bankingApp/internal/model"
	"log/slog"

	"gorm.io/gorm"
)

func createUserMigration(db *gorm.DB) {
	userExists := db.Migrator().HasTable(&model.User{})
	if !userExists {
		err := db.Migrator().CreateTable(&model.User{})
		if err != nil {
			slog.Error(err.Error())
		}
	}
}

func createAccountMigration(db *gorm.DB) {
	accountExists := db.Migrator().HasTable(&model.Account{})
	if !accountExists {
		err := db.Migrator().CreateTable(&model.Account{})
		if err != nil {
			slog.Error(err.Error())
		}
	}
}

func createTransactionMigration(db *gorm.DB) {
	transactionExists := db.Migrator().HasTable(&model.Transaction{})
	if !transactionExists {
		err := db.Migrator().CreateTable(&model.Transaction{})
		if err != nil {
			slog.Error(err.Error())
		}
	}
}
