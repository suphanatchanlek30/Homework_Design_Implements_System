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

type CalculationLogHandler struct {
	service service.CalculationLogService
}

func NewCalculationLogHandler(service service.CalculationLogService) *CalculationLogHandler {
	return &CalculationLogHandler{service: service}
}

func (h *CalculationLogHandler) List(c *fiber.Ctx) error {
	query, err := parseCalculationLogQuery(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_QUERY_PARAMETER", err.Error()))
	}

	res, err := h.service.List(c.Context(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	return c.JSON(res)
}

func (h *CalculationLogHandler) GetByCalculationID(c *fiber.Ctx) error {
	calculationID := strings.TrimSpace(c.Params("calculationId"))
	if calculationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_CALCULATION_ID", "invalid calculation ID"))
	}

	res, err := h.service.GetByCalculationID(c.Context(), calculationID)
	if err != nil {
		return calculationLogErrorResponse(c, err)
	}

	return c.JSON(res)
}

func (h *CalculationLogHandler) Replay(c *fiber.Ctx) error {
	calculationID := strings.TrimSpace(c.Params("calculationId"))
	if calculationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_CALCULATION_ID", "invalid calculation ID"))
	}

	var req dto.CalculationLogReplayRequest
	if err := c.BodyParser(&req); err != nil && !strings.Contains(err.Error(), "EOF") {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Replay(c.Context(), calculationID, req)
	if err != nil {
		return calculationLogErrorResponse(c, err)
	}

	return c.JSON(res)
}

func parseCalculationLogQuery(c *fiber.Ctx) (dto.CalculationLogQuery, error) {
	query := dto.CalculationLogQuery{}

	if value := c.Query("requestId"); value != "" {
		query.RequestID = &value
	}
	if value := c.Query("orderId"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return dto.CalculationLogQuery{}, errors.New("invalid orderId parameter")
		}
		query.OrderID = &parsed
	}
	if value := c.Query("userId"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return dto.CalculationLogQuery{}, errors.New("invalid userId parameter")
		}
		query.UserID = &parsed
	}
	if value := c.Query("promotionId"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return dto.CalculationLogQuery{}, errors.New("invalid promotionId parameter")
		}
		query.PromotionID = &parsed
	}
	if value := c.Query("createdFrom"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return dto.CalculationLogQuery{}, errors.New("invalid createdFrom parameter")
		}
		query.CreatedFrom = &parsed
	}
	if value := c.Query("createdTo"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return dto.CalculationLogQuery{}, errors.New("invalid createdTo parameter")
		}
		query.CreatedTo = &parsed
	}
	if query.CreatedFrom != nil && query.CreatedTo != nil && query.CreatedFrom.After(*query.CreatedTo) {
		return dto.CalculationLogQuery{}, errors.New("invalid date range")
	}

	page, limit, err := parsePagination(c)
	if err != nil {
		return dto.CalculationLogQuery{}, err
	}
	query.Page = page
	query.Limit = limit

	if rawSort := strings.TrimSpace(c.Query("sort")); rawSort != "" {
		sortValue, err := normalizeCalculationLogSort(rawSort)
		if err != nil {
			return dto.CalculationLogQuery{}, err
		}
		query.Sort = &sortValue
	}

	return query, nil
}

func normalizeCalculationLogSort(raw string) (string, error) {
	parts := strings.Fields(strings.ToLower(raw))
	if len(parts) == 0 || len(parts) > 2 {
		return "", errors.New("invalid sort parameter")
	}

	field := parts[0]
	allowedFields := map[string]string{
		"id":          "id",
		"request_id":  "request_id",
		"order_id":    "order_id",
		"user_id":     "user_id",
		"created_at":  "created_at",
	}

	column, ok := allowedFields[field]
	if !ok {
		return "", errors.New("invalid sort parameter")
	}

	direction := "desc"
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

func calculationLogErrorResponse(c *fiber.Ctx, err error) error {
	switch err {
	case service.ErrCalculationLogNotFound:
		return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("CALCULATION_LOG_NOT_FOUND", err.Error()))
	case service.ErrReplayModeNotSupported:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("REPLAY_MODE_NOT_SUPPORTED", err.Error()))
	case service.ErrCalculationReplayFailed:
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("CALCULATION_REPLAY_FAILED", err.Error()))
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}
}
