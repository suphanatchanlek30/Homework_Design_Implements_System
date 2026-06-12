package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/service"
)

type PricingHandler struct {
	service service.PricingService
}

// NewPricingHandler binds HTTP pricing endpoints to the pricing service.
// ผูก endpoint ด้าน pricing เข้ากับ service ที่ทำหน้าที่คำนวณราคา
func NewPricingHandler(service service.PricingService) *PricingHandler {
	return &PricingHandler{service: service}
}

// Calculate handles the final pricing endpoint used by downstream flows like order confirmation.
// รับคำขอคำนวณราคาสุดท้ายเพื่อนำผลไปใช้ต่อใน flow อื่น เช่นยืนยันคำสั่งซื้อ
func (h *PricingHandler) Calculate(c *fiber.Ctx) error {
	var req dto.PricingCalculateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Calculate(c.Context(), req)
	if err != nil {
		return pricingErrorResponse(c, err)
	}

	return c.JSON(res)
}

// Explain handles pricing requests where the caller wants the applied and skipped promotion trail.
// รับคำขอคำนวณราคาแบบที่ผู้เรียกต้องการดูเหตุผลว่าโปรไหนถูกใช้หรือถูกข้าม
func (h *PricingHandler) Explain(c *fiber.Ctx) error {
	var req dto.PricingCalculateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Explain(c.Context(), req)
	if err != nil {
		return pricingErrorResponse(c, err)
	}

	return c.JSON(res)
}

// pricingErrorResponse maps pricing-domain errors into stable HTTP responses.
// แปลง error ฝั่ง pricing ให้เป็น HTTP response ที่รูปแบบคงที่และคาดเดาได้
func pricingErrorResponse(c *fiber.Ctx, err error) error {
	switch err {
	case service.ErrEmptyOrderItems:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("EMPTY_ORDER_ITEMS", err.Error()))
	case service.ErrInvalidQuantity:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("INVALID_QUANTITY", err.Error()))
	case service.ErrProductNotFound:
		return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("PRODUCT_NOT_FOUND", err.Error()))
	case service.ErrProductInactive:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("PRODUCT_INACTIVE", err.Error()))
	case service.ErrCurrencyMismatch:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("CURRENCY_MISMATCH", err.Error()))
	case service.ErrCalculationFailed:
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("CALCULATION_FAILED", err.Error()))
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}
}
