package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	appconfig "github.com/suphanatchanlek30/homework_design_implements_system/internal/config"
	appdatabase "github.com/suphanatchanlek30/homework_design_implements_system/internal/database"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/handler"
	appmiddleware "github.com/suphanatchanlek30/homework_design_implements_system/internal/middleware"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/service"
)

// main wires the full HTTP application, dependencies, routes, and process startup.
// ประกอบแอป HTTP ทั้งระบบ รวม dependency, routes และการเริ่มต้น process
func main() {
	_ = godotenv.Load()

	config := appconfig.Load()
	db, err := appdatabase.ConnectGORM(config.MySQLDSN())
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}

	app := fiber.New()
	app.Use(appmiddleware.RequestID())
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = config.AppPort
	}

	// Repositories
	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	promotionRepo := repository.NewPromotionRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	calculationLogRepo := repository.NewCalculationLogRepository(db)

	// Services
	categoryService := service.NewCategoryService(categoryRepo)
	productService := service.NewProductService(productRepo, categoryRepo)
	promotionService := service.NewPromotionService(db, promotionRepo)
	pricingService := service.NewPricingService(db, productRepo, promotionRepo)
	orderService := service.NewOrderService(db, orderRepo, promotionRepo, pricingService)
	calculationLogService := service.NewCalculationLogService(calculationLogRepo, pricingService)

	// Handlers
	healthHandler := handler.NewHealthHandler(db)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	productHandler := handler.NewProductHandler(productService)
	promotionHandler := handler.NewPromotionHandler(promotionService)
	pricingHandler := handler.NewPricingHandler(pricingService)
	orderHandler := handler.NewOrderHandler(orderService)
	calculationLogHandler := handler.NewCalculationLogHandler(calculationLogService)

	apiV1 := app.Group("/api/v1")
	apiV1.Get("/healthz", healthHandler.Healthz)
	apiV1.Get("/readyz", healthHandler.Readyz)

	// Category Routes
	categories := apiV1.Group("/categories")
	categories.Post("", categoryHandler.Create)
	categories.Get("", categoryHandler.List)
	categories.Get("/:categoryId", categoryHandler.GetByID)
	categories.Patch("/:categoryId", categoryHandler.Update)

	// Product Routes
	products := apiV1.Group("/products")
	products.Post("", productHandler.Create)
	products.Get("", productHandler.List)
	products.Get("/:productId", productHandler.GetByID)
	products.Patch("/:productId", productHandler.Update)

	// Promotion Routes
	promotions := apiV1.Group("/promotions")
	promotions.Post("", promotionHandler.Create)
	promotions.Get("", promotionHandler.List)
	promotions.Get("/:promotionId", promotionHandler.GetByID)
	promotions.Put("/:promotionId", promotionHandler.Replace)
	promotions.Patch("/:promotionId", promotionHandler.Patch)
	promotions.Post("/:promotionId/validate", promotionHandler.Validate)
	promotions.Post("/:promotionId/activate", promotionHandler.Activate)
	promotions.Post("/:promotionId/deactivate", promotionHandler.Deactivate)
	promotions.Get("/:promotionId/usages", promotionHandler.Usages)

	// Pricing Routes
	pricing := apiV1.Group("/pricing")
	pricing.Post("/calculate", pricingHandler.Calculate)
	pricing.Post("/explain", pricingHandler.Explain)

	// Order Routes
	orders := apiV1.Group("/orders")
	orders.Post("/confirm", orderHandler.Confirm)
	orders.Get("", orderHandler.List)
	orders.Get("/:orderId", orderHandler.GetByID)

	// Audit Routes
	calculationLogs := apiV1.Group("/calculation-logs")
	calculationLogs.Get("", calculationLogHandler.List)
	calculationLogs.Get("/:calculationId", calculationLogHandler.GetByCalculationID)
	calculationLogs.Post("/:calculationId/replay", calculationLogHandler.Replay)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
