package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/service"
)

type CategoryHandler struct {
	service service.CategoryService
}

// NewCategoryHandler binds category HTTP endpoints to the category service.
// ผูก endpoint ด้าน category เข้ากับ service ที่ดูแล logic หมวดหมู่สินค้า
func NewCategoryHandler(service service.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

// Create accepts a category payload and creates a new category resource.
// รับ payload ของ category แล้วสร้าง resource หมวดหมู่ใหม่
func (h *CategoryHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Create(c.Context(), req)
	if err != nil {
		switch err {
		case service.ErrParentNotFound:
			return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("PARENT_CATEGORY_NOT_FOUND", err.Error()))
		case service.ErrCategoryAlreadyExists:
			return c.Status(fiber.StatusConflict).JSON(newErrorResponse("CATEGORY_ALREADY_EXISTS", err.Error()))
		case service.ErrInvalidCategoryName, service.ErrInvalidCategoryStatus:
			return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", err.Error()))
		default:
			return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", err.Error()))
		}
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

// List returns category summaries using validated query filters and pagination.
// คืนรายการหมวดหมู่โดยใช้ query ที่ผ่านการตรวจและข้อมูลแบ่งหน้า
func (h *CategoryHandler) List(c *fiber.Ctx) error {
	query, err := parseCategoryQuery(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_QUERY_PARAMETER", err.Error()))
	}

	res, err := h.service.List(c.Context(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	return c.JSON(res)
}

// GetByID returns one category by its path parameter ID.
// คืนข้อมูลหมวดหมู่หนึ่งรายการจาก ID ใน path parameter
func (h *CategoryHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("categoryId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_CATEGORY_ID", "invalid category ID"))
	}

	res, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == service.ErrCategoryNotFound {
			return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("CATEGORY_NOT_FOUND", err.Error()))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(newErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	return c.JSON(res)
}

// Update applies partial updates to one category resource.
// อัปเดตข้อมูลบางส่วนของ category หนึ่งรายการ
func (h *CategoryHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("categoryId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_CATEGORY_ID", "invalid category ID"))
	}

	var req dto.UpdateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", "invalid request body"))
	}

	res, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		switch err {
		case service.ErrCategoryNotFound:
			return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("CATEGORY_NOT_FOUND", err.Error()))
		case service.ErrParentNotFound:
			return c.Status(fiber.StatusNotFound).JSON(newErrorResponse("PARENT_CATEGORY_NOT_FOUND", err.Error()))
		case service.ErrCircularHierarchy:
			return c.Status(fiber.StatusUnprocessableEntity).JSON(newErrorResponse("INVALID_CATEGORY_HIERARCHY", err.Error()))
		case service.ErrCategoryUpdateConflict:
			return c.Status(fiber.StatusConflict).JSON(newErrorResponse("CATEGORY_UPDATE_CONFLICT", err.Error()))
		case service.ErrInvalidCategoryName, service.ErrInvalidCategoryStatus:
			return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", err.Error()))
		default:
			return c.Status(fiber.StatusBadRequest).JSON(newErrorResponse("INVALID_REQUEST", err.Error()))
		}
	}

	return c.JSON(res)
}

// parseCategoryQuery validates category list filters and normalizes pagination and sort values.
// ตรวจ query สำหรับ list categories และปรับค่า page, limit, sort ให้อยู่ในรูปแบบมาตรฐาน
func parseCategoryQuery(c *fiber.Ctx) (dto.CategoryQuery, error) {
	query := dto.CategoryQuery{}

	if value := c.Query("status"); value != "" {
		query.Status = &value
	}

	if value := c.Query("parentId"); value != "" {
		parsedParentID, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return dto.CategoryQuery{}, errors.New("invalid parentId parameter")
		}
		query.ParentID = &parsedParentID
	}

	if value := c.Query("keyword"); value != "" {
		query.Keyword = &value
	}

	page := 1
	if raw := c.Query("page"); raw != "" {
		parsedPage, err := strconv.Atoi(raw)
		if err != nil || parsedPage < 1 {
			return dto.CategoryQuery{}, errors.New("invalid page parameter")
		}
		page = parsedPage
	}

	limit := 10
	if raw := c.Query("limit"); raw != "" {
		parsedLimit, err := strconv.Atoi(raw)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			return dto.CategoryQuery{}, errors.New("invalid limit parameter")
		}
		limit = parsedLimit
	}

	query.Page = page
	query.Limit = limit

	if rawSort := strings.TrimSpace(c.Query("sort")); rawSort != "" {
		sort, err := normalizeCategorySort(rawSort)
		if err != nil {
			return dto.CategoryQuery{}, err
		}
		query.Sort = &sort
	}

	return query, nil
}

// normalizeCategorySort whitelists sortable category columns for safe ORDER BY generation.
// จำกัดคอลัมน์ที่ใช้ sort ได้เพื่อสร้าง ORDER BY อย่างปลอดภัย
func normalizeCategorySort(raw string) (string, error) {
	parts := strings.Fields(strings.ToLower(raw))
	if len(parts) == 0 || len(parts) > 2 {
		return "", errors.New("invalid sort parameter")
	}

	field := parts[0]
	allowedFields := map[string]string{
		"id":         "id",
		"name":       "name",
		"parent_id":  "parent_id",
		"status":     "status",
		"created_at": "created_at",
		"updated_at": "updated_at",
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
