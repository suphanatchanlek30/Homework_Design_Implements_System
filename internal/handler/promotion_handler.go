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

type PromotionHandler struct {
	service service.PromotionService
}

func NewPromotionHandler(service service.PromotionService) *PromotionHandler {
	return &PromotionHandler{service: service}
}

func (h *PromotionHandler) Create(c *fiber.Ctx) error {
	var req dto.PromotionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Create(c.Context(), req)
	if err != nil {
		return promotionErrorResponse(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

func (h *PromotionHandler) List(c *fiber.Ctx) error {
	query, err := parsePromotionListQuery(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_QUERY_PARAMETER", err.Error()))
	}

	res, err := h.service.List(c.Context(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	return c.JSON(res)
}

func (h *PromotionHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("promotionId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_PROMOTION_ID", "invalid promotion ID"))
	}

	res, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return promotionErrorResponse(c, err)
	}

	return c.JSON(res)
}

func (h *PromotionHandler) Replace(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("promotionId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_PROMOTION_ID", "invalid promotion ID"))
	}

	var req dto.PromotionReplaceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Replace(c.Context(), id, req)
	if err != nil {
		return promotionErrorResponse(c, err)
	}

	return c.JSON(res)
}

func (h *PromotionHandler) Patch(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("promotionId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_PROMOTION_ID", "invalid promotion ID"))
	}

	var req dto.PromotionPatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Patch(c.Context(), id, req)
	if err != nil {
		return promotionErrorResponse(c, err)
	}

	return c.JSON(res)
}

func (h *PromotionHandler) Validate(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("promotionId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_PROMOTION_ID", "invalid promotion ID"))
	}

	var req dto.PromotionValidateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Validate(c.Context(), id, req)
	if err != nil {
		return promotionErrorResponse(c, err)
	}

	return c.JSON(res)
}

func (h *PromotionHandler) Activate(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("promotionId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_PROMOTION_ID", "invalid promotion ID"))
	}

	var req dto.PromotionActivateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Activate(c.Context(), id, req)
	if err != nil {
		return promotionErrorResponse(c, err)
	}

	return c.JSON(res)
}

func (h *PromotionHandler) Deactivate(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("promotionId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_PROMOTION_ID", "invalid promotion ID"))
	}

	var req dto.PromotionDeactivateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Deactivate(c.Context(), id, req)
	if err != nil {
		return promotionErrorResponse(c, err)
	}

	return c.JSON(res)
}

func (h *PromotionHandler) Usages(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("promotionId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_PROMOTION_ID", "invalid promotion ID"))
	}

	query, err := parsePromotionUsageQuery(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_QUERY_PARAMETER", err.Error()))
	}

	res, err := h.service.Usages(c.Context(), id, query)
	if err != nil {
		return promotionErrorResponse(c, err)
	}

	return c.JSON(res)
}

func parsePromotionListQuery(c *fiber.Ctx) (dto.PromotionListQuery, error) {
	query := dto.PromotionListQuery{}
	if value := c.Query("status"); value != "" {
		if !isPromotionStatus(value) {
			return dto.PromotionListQuery{}, errors.New("invalid status parameter")
		}
		query.Status = &value
	}
	if value := c.Query("scope"); value != "" {
		if !isPromotionScope(value) {
			return dto.PromotionListQuery{}, errors.New("invalid scope parameter")
		}
		query.Scope = &value
	}
	if value := c.Query("actionType"); value != "" {
		query.ActionType = &value
	}
	if value := c.Query("code"); value != "" {
		query.Code = &value
	}
	if value := c.Query("activeAt"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return dto.PromotionListQuery{}, err
		}
		query.ActiveAt = &parsed
	}
	page, limit, err := parsePagination(c)
	if err != nil {
		return dto.PromotionListQuery{}, err
	}
	query.Page = page
	query.Limit = limit
	if rawSort := strings.TrimSpace(c.Query("sort")); rawSort != "" {
		sortValue, err := normalizePromotionSort(rawSort)
		if err != nil {
			return dto.PromotionListQuery{}, err
		}
		query.Sort = &sortValue
	}
	return query, nil
}

func parsePromotionUsageQuery(c *fiber.Ctx) (dto.PromotionUsageQuery, error) {
	query := dto.PromotionUsageQuery{}
	if value := c.Query("userId"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return dto.PromotionUsageQuery{}, err
		}
		query.UserID = &parsed
	}
	if value := c.Query("from"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return dto.PromotionUsageQuery{}, err
		}
		query.From = &parsed
	}
	if value := c.Query("to"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return dto.PromotionUsageQuery{}, err
		}
		query.To = &parsed
	}
	page, limit, err := parsePagination(c)
	if err != nil {
		return dto.PromotionUsageQuery{}, err
	}
	query.Page = page
	query.Limit = limit
	return query, nil
}

func parsePagination(c *fiber.Ctx) (int, int, error) {
	page := 1
	limit := 10
	if raw := c.Query("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return 0, 0, errors.New("invalid page parameter")
		}
		page = parsed
	}
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			return 0, 0, errors.New("invalid limit parameter")
		}
		limit = parsed
	}
	return page, limit, nil
}

func promotionErrorResponse(c *fiber.Ctx, err error) error {
	switch err {
	case service.ErrPromotionNotFound:
		return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("PROMOTION_NOT_FOUND", err.Error()))
	case service.ErrPromotionCodeAlreadyExists:
		return c.Status(fiber.StatusConflict).JSON(newErrorResponse("PROMOTION_CODE_ALREADY_EXISTS", err.Error()))
	case service.ErrPromotionVersionConflict:
		return c.Status(fiber.StatusConflict).JSON(newErrorResponse("PROMOTION_VERSION_CONFLICT", err.Error()))
	case service.ErrInvalidPromotionConfig:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("INVALID_PROMOTION_CONFIG", err.Error()))
	case service.ErrActionStrategyNotSupported:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("ACTION_STRATEGY_NOT_SUPPORTED", err.Error()))
	case service.ErrTargetRequired:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("TARGET_REQUIRED", err.Error()))
	case service.ErrFieldNotPatchable:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("FIELD_NOT_PATCHABLE", err.Error()))
	case service.ErrPromotionAlreadyInactive:
		return c.Status(fiber.StatusConflict).JSON(newErrorResponse("PROMOTION_ALREADY_INACTIVE", err.Error()))
	case service.ErrPromotionAlreadyExpired:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("PROMOTION_ALREADY_EXPIRED", err.Error()))
	case service.ErrPromotionConfigurationInvalid:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("PROMOTION_CONFIGURATION_INVALID", err.Error()))
	default:
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", err.Error()))
	}
}

func isPromotionStatus(value string) bool {
	switch value {
	case "DRAFT", "ACTIVE", "INACTIVE", "EXPIRED":
		return true
	default:
		return false
	}
}

func isPromotionScope(value string) bool {
	switch value {
	case "ITEM", "CART", "COUPON", "SHIPPING":
		return true
	default:
		return false
	}
}

func normalizePromotionSort(raw string) (string, error) {
	parts := strings.Fields(strings.ToLower(raw))
	if len(parts) == 0 || len(parts) > 2 {
		return "", errors.New("invalid sort parameter")
	}

	allowedFields := map[string]string{
		"id":         "id",
		"code":       "code",
		"name":       "name",
		"scope":      "scope",
		"status":     "status",
		"priority":   "priority",
		"starts_at":  "starts_at",
		"ends_at":    "ends_at",
		"created_at": "created_at",
		"updated_at": "updated_at",
		"version":    "version",
	}

	column, ok := allowedFields[parts[0]]
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
