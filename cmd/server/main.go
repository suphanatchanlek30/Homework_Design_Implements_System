package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	appconfig "github.com/suphanatchanlek30/homework_design_implements_system/internal/config"
	appdatabase "github.com/suphanatchanlek30/homework_design_implements_system/internal/database"
)

func main() {
	_ = godotenv.Load()

	config := appconfig.Load()
	db, err := appdatabase.OpenMySQL(config.MySQLDSN())
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer db.Close()

	app := fiber.New()
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = config.AppPort
	}

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unhealthy",
				"db":     "down",
				"error":   err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status": "ok",
			"db":     "up",
		})
	})

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("listen: %v", err)
	}
}