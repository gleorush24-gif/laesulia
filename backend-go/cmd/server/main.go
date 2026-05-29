package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/laesulia/api/internal/database"
	"github.com/laesulia/api/internal/handlers"
	"github.com/laesulia/api/internal/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	if err := database.MigrateBounty(db); err != nil {
		log.Fatalf("Failed to run bounty migrations: %v", err)
	}
	database.MigrateBase64(db)
	database.MigrateAdmin(db)
	database.MigratePhone(db)
	database.MigrateTreasure(db)

	r := gin.Default()
	r.Use(middleware.CORS())

	// Public routes
	auth := handlers.NewAuthHandler(db)
	r.POST("/api/v1/auth/register", auth.Register)
	r.POST("/api/v1/auth/login", auth.Login)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "laesulia-api"})
	})
	// Serve uploaded files
	r.Static("/uploads", "/app/uploads")

	// Public — anyone can read locations (no login needed)
	loc := handlers.NewLocationHandler(db)
	r.GET("/api/v1/locations", loc.List)
	r.GET("/api/v1/locations/:id", loc.Get)
	// Public bounties — anyone can see open bounty pins
	bounty := handlers.NewBountyHandler(db)
	treasure := handlers.NewTreasureHandler(db)
	r.GET("/api/v1/bounties", bounty.List)
	r.GET("/api/v1/treasure", treasure.List)

	// Protected — must be logged in
	api := r.Group("/api/v1", middleware.Auth())
	{
		// Locations
		api.POST("/locations", loc.Create)
		api.PUT("/locations/:id", loc.Update)
		api.DELETE("/locations/:id", loc.Delete)
		api.POST("/locations/:id/upvote", loc.Upvote)

		// Bounties
		bounty := handlers.NewBountyHandler(db)

		api.POST("/bounties/:id/claim", bounty.Claim)
		api.POST("/bounties/:id/submit", bounty.Submit)
		api.POST("/bounties/:id/upload", bounty.UploadFile)
		api.GET("/wallet", bounty.GetWallet)

		// Admin bounty routes
		api.POST("/bounties", bounty.Create)
		api.POST("/bounties/:id/approve", bounty.Approve)
		api.DELETE("/bounties/:id", bounty.Delete)
		api.GET("/admin/bounties/submitted", bounty.GetSubmitted)
		api.GET("/admin/bounties/:id/files", bounty.GetFiles)
                api.POST("/treasure", treasure.Create)
                api.POST("/treasure/:id/questions", treasure.AddQuestion)
                api.GET("/treasure/:id/questions", treasure.GetQuestions)
                api.POST("/treasure/:id/start", treasure.StartAttempt)
                api.POST("/treasure/:id/answer", treasure.SubmitAnswer)
                api.POST("/treasure/:id/bet", treasure.SubmitBet)
                api.POST("/treasure/:id/resolve-bet", treasure.ResolveBet)
                api.GET("/treasure/:id/finalists", treasure.GetFinalists)
                api.POST("/treasure/:id/winner", treasure.DeclareWinner)
                api.POST("/treasure/:id/reset", treasure.ResetAttempt)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🗺️  Laesulia API running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
