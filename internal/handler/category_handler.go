package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/service"
)

type CategoryHandler struct {
	service service.CategoryService
}

func NewCategoryHandler(service service.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

func (h *CategoryHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	res, err := h.service.Create(c.Context(), req)
	if err != nil {
		switch err {
		case service.ErrParentNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		case service.ErrCategoryAlreadyExists:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		default:
			// Handle validation errors (simple string matching for now or specific error types)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

func (h *CategoryHandler) List(c *fiber.Ctx) error {
	var query dto.CategoryQuery
	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid query parameters"})
	}

	res, err := h.service.List(c.Context(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(res)
}

func (h *CategoryHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("categoryId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category ID"})
	}

	res, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == service.ErrCategoryNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(res)
}

func (h *CategoryHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("categoryId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category ID"})
	}

	var req dto.UpdateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	res, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		switch err {
		case service.ErrCategoryNotFound, service.ErrParentNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		case service.ErrCircularHierarchy:
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.JSON(res)
}
