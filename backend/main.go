package main

import (
	"log"
	"os"
	"os/exec"
	"steganography-backend/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Check if LAME encoder is available
	if err := checkLAMEAvailability(); err != nil {
		log.Fatalf("LAME encoder not found: %v", err)
	}
	log.Printf("✓ LAME encoder found and ready for MP3 encoding")

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	config.ExposeHeaders = []string{"X-Stego-PSNR", "X-Stego-Message", "Content-Disposition"}
	config.AllowCredentials = true
	router.Use(cors.New(config))

	stegoHandler := handlers.NewStegoHandler()

	// API Routes
	api := router.Group("/api/v1")
	{
		api.GET("/health", stegoHandler.HealthCheck)

		stego := api.Group("/stego")
		{
			stego.POST("/insert", stegoHandler.InsertMessage)
			stego.POST("/extract", stegoHandler.ExtractMessage)
		}
	}

	// Note: Files are now streamed directly from endpoints, no separate download route needed

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("API endpoints:")
	log.Printf("  POST /api/v1/stego/insert  - Insert secret message into MP3 (returns stego MP3)")
	log.Printf("  POST /api/v1/stego/extract - Extract secret message from MP3 (returns secret file)")
	log.Printf("  GET  /api/v1/health        - Health check")
	log.Printf("")
	log.Printf("Features:")
	log.Printf("  • MP3 input/output with metadata preservation")
	log.Printf("  • LSB steganography on PCM samples")
	log.Printf("  • Vigenère cipher encryption")
	log.Printf("  • PSNR quality assessment (returned in X-Stego-PSNR header)")
	log.Printf("  • Direct streaming (no disk storage)")
	log.Printf("")
	log.Printf("Requirements: LAME encoder must be installed for MP3 encoding")

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// checkLAMEAvailability verifies that LAME encoder is installed and accessible
func checkLAMEAvailability() error {
	cmd := exec.Command("lame", "--version")
	return cmd.Run()
}
