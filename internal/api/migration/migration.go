package migration

import (
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) {
	createUserMigration(db)
	createAccountMigration(db)
	createTransactionMigration(db)
	addPaymentReferenceColumn(db)
}
