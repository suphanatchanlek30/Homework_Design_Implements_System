package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	appconfig "github.com/suphanatchanlek30/homework_design_implements_system/internal/config"
	appdatabase "github.com/suphanatchanlek30/homework_design_implements_system/internal/database"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/handler"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/service"
)

func main() {
	_ = godotenv.Load()

	config := appconfig.Load()
	db, err := appdatabase.ConnectGORM(config.MySQLDSN())
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}

	app := fiber.New()
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = config.AppPort
	}

	// Repositories
	categoryRepo := repository.NewCategoryRepository(db)

	// Services
	categoryService := service.NewCategoryService(categoryRepo)

	// Handlers
	healthHandler := handler.NewHealthHandler(db)
	categoryHandler := handler.NewCategoryHandler(categoryService)

	apiV1 := app.Group("/api/v1")
	apiV1.Get("/healthz", healthHandler.Healthz)
	apiV1.Get("/readyz", healthHandler.Readyz)

	// Category Routes
	categories := apiV1.Group("/categories")
	categories.Post("/", categoryHandler.Create)
	categories.Get("/", categoryHandler.List)
	categories.Get("/:categoryId", categoryHandler.GetByID)
	categories.Patch("/:categoryId", categoryHandler.Update)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("listen: %v", err)
	}
}