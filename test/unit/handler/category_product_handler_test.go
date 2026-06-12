package handler_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/handler"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/service"
)

type mockCategoryService struct {
	createFn func(req dto.CreateCategoryRequest) (*dto.CategoryResponse, error)
	listFn   func(query dto.CategoryQuery) (*dto.CategoryListResponse, error)
	getFn    func(id uint64) (*dto.CategoryResponse, error)
	updateFn func(id uint64, req dto.UpdateCategoryRequest) (*dto.CategoryResponse, error)
}

// Create returns the mocked category-create result used by handler tests.
// คืนผลสร้าง category แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockCategoryService) Create(_ context.Context, req dto.CreateCategoryRequest) (*dto.CategoryResponse, error) {
	return m.createFn(req)
}

// Update returns the mocked category-update result used by handler tests.
// คืนผลอัปเดต category แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockCategoryService) Update(_ context.Context, id uint64, req dto.UpdateCategoryRequest) (*dto.CategoryResponse, error) {
	return m.updateFn(id, req)
}

// GetByID returns the mocked category-detail result used by handler tests.
// คืนผล category detail แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockCategoryService) GetByID(_ context.Context, id uint64) (*dto.CategoryResponse, error) {
	return m.getFn(id)
}

// List returns the mocked category-list result used by handler tests.
// คืนผล list categories แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockCategoryService) List(_ context.Context, query dto.CategoryQuery) (*dto.CategoryListResponse, error) {
	return m.listFn(query)
}

type mockProductService struct {
	createFn func(req dto.CreateProductRequest) (*dto.ProductResponse, error)
	listFn   func(query dto.ProductQuery) (*dto.ProductListResponse, error)
	getFn    func(id uint64) (*dto.ProductResponse, error)
	updateFn func(id uint64, req dto.UpdateProductRequest) (*dto.ProductResponse, error)
}

// Create returns the mocked product-create result used by handler tests.
// คืนผลสร้างสินค้าแบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockProductService) Create(_ context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error) {
	return m.createFn(req)
}

// Update returns the mocked product-update result used by handler tests.
// คืนผลอัปเดตสินค้าแบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockProductService) Update(_ context.Context, id uint64, req dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	return m.updateFn(id, req)
}

// GetByID returns the mocked product-detail result used by handler tests.
// คืนผล product detail แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockProductService) GetByID(_ context.Context, id uint64) (*dto.ProductResponse, error) {
	return m.getFn(id)
}

// List returns the mocked product-list result used by handler tests.
// คืนผล list products แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockProductService) List(_ context.Context, query dto.ProductQuery) (*dto.ProductListResponse, error) {
	return m.listFn(query)
}

// TestCategoryCreate_Conflict verifies duplicate categories return HTTP 409.
// ตรวจว่าการสร้าง category ซ้ำจะตอบกลับ HTTP 409
func TestCategoryCreate_Conflict(t *testing.T) {
	app := fiber.New()
	svc := &mockCategoryService{
		createFn: func(req dto.CreateCategoryRequest) (*dto.CategoryResponse, error) {
			return nil, service.ErrCategoryAlreadyExists
		},
	}

	app.Post("/api/v1/categories", handler.NewCategoryHandler(svc).Create)

	req := httptest.NewRequest("POST", "/api/v1/categories", strings.NewReader(`{"name":"Electronics","status":"ACTIVE"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "CATEGORY_ALREADY_EXISTS", body.Error.Code)
}

// TestCategoryList_InvalidQuery verifies invalid category list queries return HTTP 400.
// ตรวจว่า query ที่ไม่ถูกต้องของ category list จะตอบกลับ HTTP 400
func TestCategoryList_InvalidQuery(t *testing.T) {
	app := fiber.New()
	svc := &mockCategoryService{
		listFn: func(query dto.CategoryQuery) (*dto.CategoryListResponse, error) {
			return &dto.CategoryListResponse{}, nil
		},
	}

	app.Get("/api/v1/categories", handler.NewCategoryHandler(svc).List)

	req := httptest.NewRequest("GET", "/api/v1/categories?page=abc", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "INVALID_QUERY_PARAMETER", body.Error.Code)
}

// TestCategoryUpdate_Conflict verifies conflicting category updates return HTTP 409.
// ตรวจว่าการอัปเดต category ที่ชนกันจะตอบกลับ HTTP 409
func TestCategoryUpdate_Conflict(t *testing.T) {
	app := fiber.New()
	svc := &mockCategoryService{
		updateFn: func(id uint64, req dto.UpdateCategoryRequest) (*dto.CategoryResponse, error) {
			return nil, service.ErrCategoryUpdateConflict
		},
	}

	app.Patch("/api/v1/categories/:categoryId", handler.NewCategoryHandler(svc).Update)

	req := httptest.NewRequest("PATCH", "/api/v1/categories/1", strings.NewReader(`{"name":"Electronics","parentId":2}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "CATEGORY_UPDATE_CONFLICT", body.Error.Code)
}

// TestProductCreate_Conflict verifies duplicate product SKUs return HTTP 409.
// ตรวจว่าการสร้างสินค้าที่ SKU ซ้ำจะตอบกลับ HTTP 409
func TestProductCreate_Conflict(t *testing.T) {
	app := fiber.New()
	svc := &mockProductService{
		createFn: func(req dto.CreateProductRequest) (*dto.ProductResponse, error) {
			return nil, service.ErrSKUAlreadyExists
		},
	}

	app.Post("/api/v1/products", handler.NewProductHandler(svc).Create)

	req := httptest.NewRequest("POST", "/api/v1/products", strings.NewReader(`{"sku":"PRODUCT-001","name":"Product 1","categoryId":1,"priceAmount":100000,"currency":"THB","status":"ACTIVE"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "SKU_ALREADY_EXISTS", body.Error.Code)
}

// TestProductUpdate_CategoryNotFound verifies updates with missing categories return HTTP 404.
// ตรวจว่าการอัปเดตสินค้าด้วย category ที่ไม่มีอยู่จะตอบกลับ HTTP 404
func TestProductUpdate_CategoryNotFound(t *testing.T) {
	app := fiber.New()
	svc := &mockProductService{
		updateFn: func(id uint64, req dto.UpdateProductRequest) (*dto.ProductResponse, error) {
			return nil, service.ErrCategoryNotFound
		},
	}

	app.Patch("/api/v1/products/:productId", handler.NewProductHandler(svc).Update)

	req := httptest.NewRequest("PATCH", "/api/v1/products/1", strings.NewReader(`{"categoryId":999}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "CATEGORY_NOT_FOUND", body.Error.Code)
}

// TestProductList_InvalidSort verifies unsafe sort expressions return HTTP 400.
// ตรวจว่า sort ที่ไม่ปลอดภัยหรือไม่ถูกต้องจะตอบกลับ HTTP 400
func TestProductList_InvalidSort(t *testing.T) {
	app := fiber.New()
	svc := &mockProductService{
		listFn: func(query dto.ProductQuery) (*dto.ProductListResponse, error) {
			return &dto.ProductListResponse{}, nil
		},
	}

	app.Get("/api/v1/products", handler.NewProductHandler(svc).List)

	req := httptest.NewRequest("GET", "/api/v1/products?sort=DROP%20TABLE", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "INVALID_QUERY_PARAMETER", body.Error.Code)
}
