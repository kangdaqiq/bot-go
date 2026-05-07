package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"bot_wa/config"
	"bot_wa/handlers"
	"bot_wa/jobs"
	"bot_wa/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file, using environment variables")
	}

	// Initialize Database
	config.ConnectDB()

	// Initialize WhatsApp Client configuration
	config.InitWhatsApp()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "WhatsApp Bot Webhook",
	})

	// Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Use(logger.New(logger.Config{
		Format: "${time} - ${method} ${path}\n",
	}))

	// Routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "running",
			"service":   "WhatsApp Bot Absensi (Go)",
			"version":   "1.0.0",
		})
	})

	webhookGroup := app.Group("/webhook")
	webhookGroup.Post("/", handlers.HandleWebhook)
	webhookGroup.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "WhatsApp Bot Webhook is running",
		})
	})

	// Setup 404 handler
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Not Found",
			"path":  c.Path(),
		})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	// Start session cleanup
	services.StartSessionCleanup()

	// Start reminder scheduler
	go jobs.StartReminderScheduler()

	go func() {
		log.Printf("✅ Server running on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Panic(err)
		}
	}()

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Gracefully shutting down...")
	_ = app.Shutdown()
	log.Println("Server closed")
}
