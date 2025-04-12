package models

import (
	"time"

	"gorm.io/gorm"
)

type Portfolio struct {
	gorm.Model
	UserID        uint   `gorm:"index"`
	Symbol        string `gorm:"index"`
	Quantity      int
	PurchasePrice float64
	Date          time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

type Transaction struct {
	gorm.Model
	UserID    uint
	Type      string // buy/sell
	Symbol    string
	Quantity  int
	Price     float64
	Timestamp time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
