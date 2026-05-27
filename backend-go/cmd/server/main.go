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
	r := gin.Default()
	r.Use(middleware.CORS())
	auth := handlers.NewAuthHandler(db)
	r.POST("/api/v1/auth/register", auth.Register)
	r.POST("/api/v1/auth/login", auth.Login)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "laesulia-api"})
	})
	loc    := handlers.NewLocationHandler(db)
	bounty := handlers.NewBountyHandler(db)
	r.GET("/api/v1/locations", loc.List)
	r.GET("/api/v1/locations/:id", loc.Get)
	r.GET("/api/v1/bounties", bounty.List)
	api := r.Group("/api/v1", middleware.Auth())
	{
		api.POST("/locations", loc.Create)
		api.PUT("/locations/:id", loc.Update)
		api.DELETE("/locations/:id", loc.Delete)
		api.POST("/locations/:id/upvote", loc.Upvote)
		api.POST("/bounties/:id/claim",      bounty.Claim)
		api.POST("/bounties/:id/submit",     bounty.Submit)
		api.POST("/bounties/:id/upload",     bounty.UploadFile)
		api.GET("/wallet",                   bounty.GetWallet)
		api.POST("/bounties",                bounty.Create)
		api.POST("/bounties/:id/approve",    bounty.Approve)
		api.DELETE("/bounties/:id",          bounty.Delete)
		api.GET("/admin/bounties/submitted", bounty.GetSubmitted)
		api.GET("/admin/bounties/:id/files", bounty.GetFiles)
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
