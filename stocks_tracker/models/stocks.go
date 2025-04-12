package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

type StockPrice struct {
	gorm.Model
	Symbol    string
	Price     float64
	Timestamp time.Time
}
