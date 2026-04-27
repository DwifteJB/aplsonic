package db

import (
	"github.com/DwifteJB/aplsonic/src/db/schema"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(dsn string) error {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	DB = db

	err = db.AutoMigrate(schema.AllModels...)

	if err != nil {
		return err
	}

	return nil
}
