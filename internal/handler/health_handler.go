package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler binds health endpoints to the shared database dependency.
// ผูก endpoint ด้าน health เข้ากับ dependency ฐานข้อมูลที่ใช้ตรวจความพร้อม
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Healthz checks if the application process is running.
// ตรวจว่า process ของแอปยังทำงานอยู่หรือไม่
func (h *HealthHandler) Healthz(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(dto.HealthResponse{
		Status: "UP",
	})
}

// Readyz checks if the service and its dependencies are ready.
// ตรวจว่าบริการและ dependency สำคัญพร้อมให้ใช้งานหรือไม่
func (h *HealthHandler) Readyz(c *fiber.Ctx) error {
	res := dto.ReadyResponse{
		Status: "READY",
		Dependencies: dto.ReadyDependencies{
			MySQL: "UP",
		},
	}

	if h.db == nil {
		res.Status = "NOT_READY"
		res.Dependencies.MySQL = "DOWN"
		return c.Status(fiber.StatusServiceUnavailable).JSON(res)
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		res.Status = "NOT_READY"
		res.Dependencies.MySQL = "DOWN"
		return c.Status(fiber.StatusServiceUnavailable).JSON(res)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		res.Status = "NOT_READY"
		res.Dependencies.MySQL = "DOWN"
		return c.Status(fiber.StatusServiceUnavailable).JSON(res)
	}

	// Redis is not configured in this project, so we skip it or omit it.
	// Since the DTO has omitempty for Redis, it won't show up if we don't set it.

	return c.Status(fiber.StatusOK).JSON(res)
}
