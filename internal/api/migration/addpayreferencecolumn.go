package migration

import (
	"bankingApp/internal/model"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

const paymentReference = "payment_reference"

type paymentReferenceMigration struct {
	model      interface{}
	tableName  string
	column     string
	columnType string
}

func (p *paymentReferenceMigration) addPayReferenceColumn(db *gorm.DB) error {
	if db.Migrator().HasColumn(model.Transaction{}, paymentReference) {
		return nil
	}

	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s AFTER `account_id`", p.tableName, p.column, p.columnType)
	if err := db.Exec(query).Error; err != nil {
		return err
	}

	if err := db.Migrator().AddColumn(p.model, p.column); err != nil {
		return err
	}

	return nil
}

func addPaymentReferenceColumn(db *gorm.DB) {
	p := &paymentReferenceMigration{
		model:      model.Transaction{},
		tableName:  "tbl_transaction",
		column:     paymentReference,
		columnType: "varchar(255)",
	}
	err := p.addPayReferenceColumn(db)
	if err != nil {
		slog.Error(err.Error())
	}
}
