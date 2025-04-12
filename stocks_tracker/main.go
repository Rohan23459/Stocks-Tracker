package main

import (
	"log"
	"os"
	"stocks-tracker/config"
	"stocks-tracker/handlers"
	"stocks-tracker/middleware.go"
	"stocks-tracker/models"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	log.Println("JWT_SECRET:", os.Getenv("JWT_SECRET"))
	// Initialize PostgreSQL and Redis connections.
	config.InitDB()
	config.InitRedis()

	// Get underlying SQL DB and close it properly
	sqlDB, err := config.DB.DB()
	if err != nil {
		log.Fatal("Failed to get database instance: ", err)
	}
	defer sqlDB.Close()

	// Auto-migrate models.
	if err := config.DB.AutoMigrate(&models.User{}, &models.Portfolio{}, &models.StockPrice{}, &models.Transaction{}); err != nil {
		log.Fatal("Failed to migrate models:", err)
	}

	router := gin.Default()

	// Public routes
	router.POST("/signup", handlers.Signup)
	router.POST("/login", handlers.Login)

	// Protected routes
	auth := router.Group("/")
	auth.Use(middleware.JWTAuth())
	{
		auth.POST("/stocks", handlers.AddStock)
		auth.GET("/portfolio", handlers.GetPortfolio)
		auth.PUT("/stocks/:id", handlers.UpdateStock)
		auth.DELETE("/stocks/:id", handlers.DeleteStock)
		auth.GET("/prices/:symbol", handlers.GetStockPrice)
		auth.GET("/history/:symbol", handlers.GetHistoricalData)
	}

	router.Run(":8080")
}
