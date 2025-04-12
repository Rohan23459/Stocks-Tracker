package database

import (
	"fmt"
	"reflect"
	"stocks-tracker/config"
	"stocks-tracker/models"
)

// DB is the global PostgreSQL database connection.
// var DB *gorm.DB

func autoMigrate() {
	config.DB.AutoMigrate(
		&models.User{},
		&models.Portfolio{},
		&models.Transaction{},
		&models.StockPrice{},
	)
}

var (
	ErrInvalidTransaction = fmt.Errorf("invalid transaction")
	ErrInvalidData        = fmt.Errorf("invalid data, expected slice")
)

func CreateInBatches(data interface{}, batchSize int) error {
	if batchSize <= 0 {
		return ErrInvalidTransaction
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if tx.Error != nil {
		return tx.Error
	}

	slice := reflect.ValueOf(data)
	if slice.Kind() != reflect.Slice {
		return ErrInvalidData
	}

	total := slice.Len()
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		chunk := slice.Slice(i, end).Interface()
		if err := tx.Create(chunk).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("batch insert failed: %w", err)
		}
	}

	return tx.Commit().Error
}
