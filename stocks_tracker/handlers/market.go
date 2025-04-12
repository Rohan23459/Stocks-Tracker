package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"stocks-tracker/config"
	"stocks-tracker/database"
	"stocks-tracker/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	cacheExpiration = 5 * time.Minute
)

type AlphaVantageResponse struct {
	GlobalQuote struct {
		Price string `json:"05. price"`
	} `json:"Global Quote"`
	TimeSeriesDaily map[string]struct {
		Open   string `json:"1. open"`
		High   string `json:"2. high"`
		Low    string `json:"3. low"`
		Close  string `json:"4. close"`
		Volume string `json:"5. volume"`
	} `json:"Time Series (Daily)"`
}

func GetStockPrice(c *gin.Context) {
	symbol := c.Param("symbol")
	ctx := context.Background()

	// Check Redis cache first
	cachedPrice, err := config.Rdb.Get(ctx, fmt.Sprintf("stock:%s:price", symbol)).Result()
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"symbol": symbol, "price": cachedPrice})
		return
	}

	// Fetch from Alpha Vantage API
	apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", symbol, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to fetch stock data"})
		return
	}
	defer resp.Body.Close()

	var result AlphaVantageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse stock data"})
		return
	}

	if result.GlobalQuote.Price == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock not found"})
		return
	}

	// Cache the price in Redis
	err = config.Rdb.Set(ctx, fmt.Sprintf("stock:%s:price", symbol), result.GlobalQuote.Price, cacheExpiration).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cache price"})
		return
	}
	PriceStr := result.GlobalQuote.Price
	PriceFloat, err := strconv.ParseFloat(PriceStr, 64) // 64-bit float
	if err != nil {
		fmt.Println("Error converting string to float:", err)
		return
	}
	// Store in PostgreSQL
	priceEntry := models.StockPrice{
		Symbol:    symbol,
		Price:     PriceFloat,
		Timestamp: time.Now(),
	}
	config.DB.Create(&priceEntry)

	c.JSON(http.StatusOK, gin.H{"symbol": symbol, "price": result.GlobalQuote.Price})
}

func GetHistoricalData(c *gin.Context) {
	symbol := c.Param("symbol")
	ctx := context.Background()

	// Check Redis cache
	cachedData, err := config.Rdb.Get(ctx, fmt.Sprintf("stock:%s:history", symbol)).Result()
	if err == nil {
		var historicalData []models.StockPrice
		json.Unmarshal([]byte(cachedData), &historicalData)
		c.JSON(http.StatusOK, historicalData)
		return
	}

	// Fetch from Alpha Vantage API
	apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&apikey=%s", symbol, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to fetch historical data"})
		return
	}
	defer resp.Body.Close()

	var result AlphaVantageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse historical data"})
		return
	}

	if len(result.TimeSeriesDaily) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Historical data not found"})
		return
	}

	// Process and store historical data

	var historicalData []models.StockPrice
	for date, data := range result.TimeSeriesDaily {
		closePriceStr := data.Close
		closePriceFloat, err := strconv.ParseFloat(closePriceStr, 64) // 64-bit float
		if err != nil {
			fmt.Println("Error converting string to float:", err)
			return
		}
		timestamp, _ := time.Parse("2006-01-02", date)
		entry := models.StockPrice{
			Symbol:    symbol,
			Price:     closePriceFloat,
			Timestamp: timestamp,
		}
		historicalData = append(historicalData, entry)
	}

	// Batch insert into PostgreSQL
	database.CreateInBatches(historicalData, 100)

	// Cache in Redis
	jsonData, _ := json.Marshal(historicalData)
	config.Rdb.Set(ctx, fmt.Sprintf("stock:%s:history", symbol), jsonData, 24*time.Hour)

	c.JSON(http.StatusOK, historicalData)
}
