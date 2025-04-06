package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
)

func init() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Initialize MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI"))
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	mongoClient = client
}

func main() {
	// Create a new Gin router
	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORS())

	// API routes
	api := router.Group("/api")
	{
		// Public routes
		public := api.Group("/v1")
		{
			public.POST("/auth/login", handleLogin)
			public.POST("/auth/register", handleRegister)
		}

		// Protected routes
		protected := api.Group("/v1")
		protected.Use(AuthMiddleware())
		{
			// User service routes
			protected.GET("/users/profile", handleGetUserProfile)
			protected.PUT("/users/profile", handleUpdateUserProfile)
			protected.POST("/users/change-password", handleChangePassword)

			// File service routes
			protected.POST("/files/upload", handleFileUpload)
			protected.POST("/files/upload-url", handleFileUploadFromURL)
			protected.GET("/files", handleListFiles)
			protected.GET("/files/:id", handleGetFile)
			protected.DELETE("/files/:id", handleDeleteFile)
			protected.PUT("/files/:id/hide", handleHideFile)
			protected.GET("/files/:id/download", handleDownloadFile)

			// Analytics service routes
			protected.POST("/analytics/analyze", handleAnalyzeData)
			protected.GET("/analytics/reports", handleGetReports)
			protected.GET("/analytics/reports/:id", handleGetReport)
			protected.DELETE("/analytics/reports/:id", handleDeleteReport)
		}
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

// CORS middleware
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
} 