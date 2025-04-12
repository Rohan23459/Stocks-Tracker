package handlers

import (
	"net/http"

	"stocks-tracker/config"
	"stocks-tracker/models"

	"github.com/gin-gonic/gin"
)

type StockInput struct {
	Symbol        string  `json:"symbol" binding:"required"`
	Quantity      int     `json:"quantity" binding:"required,min=1"`
	PurchasePrice float64 `json:"purchase_price" binding:"required,min=0.01"`
}

func AddStock(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var input StockInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	portfolio := models.Portfolio{
		UserID:        userID,
		Symbol:        input.Symbol,
		Quantity:      input.Quantity,
		PurchasePrice: input.PurchasePrice,
	}

	if err := tx.Create(&portfolio).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create portfolio entry"})
		return
	}

	transaction := models.Transaction{
		UserID:   userID,
		Type:     "buy",
		Symbol:   input.Symbol,
		Quantity: input.Quantity,
		Price:    input.PurchasePrice,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record transaction"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusCreated, gin.H{"message": "Stock added successfully", "id": portfolio.ID})
}

func GetPortfolio(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var portfolio []struct {
		models.Portfolio
		CurrentPrice float64 `json:"current_price"`
		ProfitLoss   float64 `json:"profit_loss"`
	}

	query := `
		SELECT p.*, s.price as current_price,
		(s.price * p.quantity - p.purchase_price * p.quantity) as profit_loss
		FROM portfolios p
		LEFT JOIN (
			SELECT symbol, price 
			FROM stock_prices 
			WHERE (symbol, timestamp) IN (
				SELECT symbol, MAX(timestamp) 
				FROM stock_prices 
				GROUP BY symbol
			)
		) s ON p.symbol = s.symbol
		WHERE p.user_id = ?
	`

	if err := config.DB.Raw(query, userID).Scan(&portfolio).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch portfolio"})
		return
	}

	c.JSON(http.StatusOK, portfolio)
}

type UpdateStockInput struct {
	Quantity      *int     `json:"quantity"`
	PurchasePrice *float64 `json:"purchase_price"`
}

func UpdateStock(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	stockID := c.Param("id")

	var input UpdateStockInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.Portfolio
	if err := config.DB.Where("id = ? AND user_id = ?", stockID, userID).First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock not found"})
		return
	}

	tx := config.DB.Begin()
	updateData := make(map[string]interface{})

	if input.Quantity != nil {
		if *input.Quantity < existing.Quantity {
			transaction := models.Transaction{
				UserID:   userID,
				Type:     "sell",
				Symbol:   existing.Symbol,
				Quantity: existing.Quantity - *input.Quantity,
				Price:    existing.PurchasePrice,
			}
			if err := tx.Create(&transaction).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record transaction"})
				return
			}
		}
		updateData["quantity"] = *input.Quantity
	}

	if input.PurchasePrice != nil {
		updateData["purchase_price"] = *input.PurchasePrice
	}

	if err := tx.Model(&models.Portfolio{}).
		Where("id = ? AND user_id = ?", stockID, userID).
		Updates(updateData).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Stock updated successfully"})
}

func DeleteStock(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	stockID := c.Param("id")

	// Verify ownership
	var portfolio models.Portfolio
	if err := config.DB.Where("id = ? AND user_id = ?", stockID, userID).First(&portfolio).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock not found"})
		return
	}

	tx := config.DB.Begin()

	if err := tx.Delete(&portfolio).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete stock"})
		return
	}

	transaction := models.Transaction{
		UserID:   userID,
		Type:     "sell",
		Symbol:   portfolio.Symbol,
		Quantity: portfolio.Quantity,
		Price:    portfolio.PurchasePrice,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record transaction"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Stock deleted successfully"})
}
