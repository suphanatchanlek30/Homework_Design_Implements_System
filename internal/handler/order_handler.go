package handler

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/service"
)

type OrderHandler struct {
	service service.OrderService
}

func NewOrderHandler(service service.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) Confirm(c *fiber.Ctx) error {
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("IDEMPOTENCY_KEY_REQUIRED", "idempotency key is required"))
	}

	var req dto.OrderConfirmRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Confirm(c.Context(), idempotencyKey, req)
	if err != nil {
		switch err {
		case service.ErrIdempotencyKeyRequired:
			return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("IDEMPOTENCY_KEY_REQUIRED", err.Error()))
		case service.ErrOrderPriceChanged:
			return c.Status(fiber.StatusConflict).JSON(newErrorResponse("ORDER_PRICE_CHANGED", err.Error()))
		case service.ErrPromotionUsageLimitReached:
			return c.Status(fiber.StatusConflict).JSON(newErrorResponse("PROMOTION_USAGE_LIMIT_REACHED", err.Error()))
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
		case service.ErrProductUnavailable:
			return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("PRODUCT_UNAVAILABLE", err.Error()))
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("ORDER_CONFIRMATION_FAILED", err.Error()))
		}
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

func (h *OrderHandler) List(c *fiber.Ctx) error {
	query, err := parseOrderQuery(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_QUERY_PARAMETER", err.Error()))
	}

	res, err := h.service.List(c.Context(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("ORDER_CONFIRMATION_FAILED", err.Error()))
	}

	return c.JSON(res)
}

func (h *OrderHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("orderId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_ORDER_ID", "invalid order ID"))
	}

	var requesterUserID *uint64
	if raw := c.Query("userId"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_QUERY_PARAMETER", "invalid userId parameter"))
		}
		requesterUserID = &parsed
	}

	res, err := h.service.GetByID(c.Context(), id, requesterUserID)
	if err != nil {
		switch err {
		case service.ErrOrderNotFound:
			return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("ORDER_NOT_FOUND", err.Error()))
		case service.ErrOrderAccessDenied:
			return c.Status(fiber.StatusForbidden).JSON(newErrorResponse("ORDER_ACCESS_DENIED", err.Error()))
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("ORDER_CONFIRMATION_FAILED", err.Error()))
		}
	}

	return c.JSON(res)
}

func parseOrderQuery(c *fiber.Ctx) (dto.OrderListQuery, error) {
	query := dto.OrderListQuery{}

	if value := c.Query("status"); value != "" {
		switch value {
		case "DRAFT", "CONFIRMED", "PAID", "CANCELLED":
			query.Status = &value
		default:
			return dto.OrderListQuery{}, errors.New("invalid status parameter")
		}
	}

	if raw := c.Query("userId"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return dto.OrderListQuery{}, errors.New("invalid userId parameter")
		}
		query.UserID = &parsed
	}

	if raw := c.Query("createdFrom"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return dto.OrderListQuery{}, errors.New("invalid createdFrom parameter")
		}
		query.CreatedFrom = &parsed
	}

	if raw := c.Query("createdTo"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return dto.OrderListQuery{}, errors.New("invalid createdTo parameter")
		}
		query.CreatedTo = &parsed
	}

	if query.CreatedFrom != nil && query.CreatedTo != nil && query.CreatedFrom.After(*query.CreatedTo) {
		return dto.OrderListQuery{}, errors.New("invalid date range")
	}

	page := 1
	if raw := c.Query("page"); raw != "" {
		parsedPage, err := strconv.Atoi(raw)
		if err != nil || parsedPage < 1 {
			return dto.OrderListQuery{}, errors.New("invalid page parameter")
		}
		page = parsedPage
	}

	limit := 10
	if raw := c.Query("limit"); raw != "" {
		parsedLimit, err := strconv.Atoi(raw)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			return dto.OrderListQuery{}, errors.New("invalid limit parameter")
		}
		limit = parsedLimit
	}

	query.Page = page
	query.Limit = limit

	if rawSort := strings.TrimSpace(c.Query("sort")); rawSort != "" {
		sort, err := normalizeOrderSort(rawSort)
		if err != nil {
			return dto.OrderListQuery{}, err
		}
		query.Sort = &sort
	}

	return query, nil
}

func normalizeOrderSort(raw string) (string, error) {
	parts := strings.Fields(strings.ToLower(raw))
	if len(parts) == 0 || len(parts) > 2 {
		return "", errors.New("invalid sort parameter")
	}

	field := parts[0]
	allowedFields := map[string]string{
		"id":             "id",
		"order_no":       "order_no",
		"user_id":        "user_id",
		"status":         "status",
		"original_total": "original_total",
		"discount_total": "discount_total",
		"final_total":    "final_total",
		"created_at":     "created_at",
		"updated_at":     "updated_at",
	}

	column, ok := allowedFields[field]
	if !ok {
		return "", errors.New("invalid sort parameter")
	}

	direction := "asc"
	if len(parts) == 2 {
		switch parts[1] {
		case "asc", "desc":
			direction = parts[1]
		default:
			return "", errors.New("invalid sort parameter")
		}
	}

	return column + " " + strings.ToUpper(direction), nil
}
