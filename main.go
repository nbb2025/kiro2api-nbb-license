package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	if os.Getenv("ADMIN_TOKEN") == "" {
		log.Fatal("ADMIN_TOKEN environment variable is required")
	}

	dataDir := "./data"
	if err := initKeys(dataDir); err != nil {
		log.Fatalf("Failed to init keys: %v", err)
	}

	dbPath := filepath.Join(dataDir, "license.db")
	if err := initDB(dbPath); err != nil {
		log.Fatalf("Failed to init db: %v", err)
	}
	defer db.Close()

	r := gin.Default()

	r.StaticFile("/", "./static/index.html")

	api := r.Group("/api/license")
	api.POST("/issue", handleIssue)

	admin := api.Group("")
	admin.Use(adminAuth())
	admin.GET("/list", handleList)
	admin.POST("/create", handleCreate)
	admin.POST("/revoke", handleRevoke)
	admin.POST("/update", handleUpdate)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("License server starting on :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
