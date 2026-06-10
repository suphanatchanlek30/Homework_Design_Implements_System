package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/service"
)

type ProductHandler struct {
	service service.ProductService
}

func NewProductHandler(service service.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

func (h *ProductHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Create(c.Context(), req)
	if err != nil {
		switch err {
		case service.ErrCategoryNotFound:
			return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("CATEGORY_NOT_FOUND", err.Error()))
		case service.ErrSKUAlreadyExists:
			return c.Status(fiber.StatusConflict).JSON(newErrorResponse("SKU_ALREADY_EXISTS", err.Error()))
		case service.ErrInvalidPriceAmount:
			return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("INVALID_PRICE_AMOUNT", err.Error()))
		case service.ErrUnsupportedCurrency:
			return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("UNSUPPORTED_CURRENCY", err.Error()))
		case service.ErrInvalidProductName, service.ErrInvalidProductSKU, service.ErrInvalidProductStatus:
			return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", err.Error()))
		default:
			return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", err.Error()))
		}
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

func (h *ProductHandler) List(c *fiber.Ctx) error {
	query, err := parseProductQuery(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_QUERY_PARAMETER", err.Error()))
	}

	res, err := h.service.List(c.Context(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	return c.JSON(res)
}

func (h *ProductHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("productId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_PRODUCT_ID", "invalid product ID"))
	}

	res, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == service.ErrProductNotFound {
			return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("PRODUCT_NOT_FOUND", err.Error()))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	return c.JSON(res)
}

func (h *ProductHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("productId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_PRODUCT_ID", "invalid product ID"))
	}

	var req dto.UpdateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		switch err {
		case service.ErrProductNotFound:
			return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("PRODUCT_NOT_FOUND", err.Error()))
		case service.ErrCategoryNotFound:
			return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("CATEGORY_NOT_FOUND", err.Error()))
		case service.ErrInvalidPriceAmount:
			return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("INVALID_PRICE_AMOUNT", err.Error()))
		case service.ErrUnsupportedCurrency:
			return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("UNSUPPORTED_CURRENCY", err.Error()))
		case service.ErrInvalidProductStatus:
			return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", err.Error()))
		default:
			return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", err.Error()))
		}
	}

	return c.JSON(res)
}

func parseProductQuery(c *fiber.Ctx) (dto.ProductQuery, error) {
	query := dto.ProductQuery{}

	if value := c.Query("status"); value != "" {
		query.Status = &value
	}

	if value := c.Query("categoryId"); value != "" {
		if _, err := strconv.ParseUint(value, 10, 64); err != nil {
			return dto.ProductQuery{}, err
		}
		query.CategoryID = &value
	}

	if value := c.Query("sku"); value != "" {
		query.SKU = &value
	}

	if value := c.Query("keyword"); value != "" {
		query.Keyword = &value
	}

	page := 1
	if raw := c.Query("page"); raw != "" {
		parsedPage, err := strconv.Atoi(raw)
		if err != nil || parsedPage < 1 {
			return dto.ProductQuery{}, errors.New("invalid page parameter")
		}
		page = parsedPage
	}

	limit := 10
	if raw := c.Query("limit"); raw != "" {
		parsedLimit, err := strconv.Atoi(raw)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			return dto.ProductQuery{}, errors.New("invalid limit parameter")
		}
		limit = parsedLimit
	}

	query.Page = page
	query.Limit = limit

	if rawSort := strings.TrimSpace(c.Query("sort")); rawSort != "" {
		sort, err := normalizeProductSort(rawSort)
		if err != nil {
			return dto.ProductQuery{}, err
		}
		query.Sort = &sort
	}

	return query, nil
}

func normalizeProductSort(raw string) (string, error) {
	parts := strings.Fields(strings.ToLower(raw))
	if len(parts) == 0 || len(parts) > 2 {
		return "", errors.New("invalid sort parameter")
	}

	field := parts[0]
	allowedFields := map[string]string{
		"id":           "id",
		"sku":          "sku",
		"name":         "name",
		"category_id":  "category_id",
		"price_amount": "price_amount",
		"status":       "status",
		"created_at":   "created_at",
		"updated_at":   "updated_at",
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
