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

func (m *mockCategoryService) Create(_ context.Context, req dto.CreateCategoryRequest) (*dto.CategoryResponse, error) {
	return m.createFn(req)
}

func (m *mockCategoryService) Update(_ context.Context, id uint64, req dto.UpdateCategoryRequest) (*dto.CategoryResponse, error) {
	return m.updateFn(id, req)
}

func (m *mockCategoryService) GetByID(_ context.Context, id uint64) (*dto.CategoryResponse, error) {
	return m.getFn(id)
}

func (m *mockCategoryService) List(_ context.Context, query dto.CategoryQuery) (*dto.CategoryListResponse, error) {
	return m.listFn(query)
}

type mockProductService struct {
	createFn func(req dto.CreateProductRequest) (*dto.ProductResponse, error)
	listFn   func(query dto.ProductQuery) (*dto.ProductListResponse, error)
	getFn    func(id uint64) (*dto.ProductResponse, error)
	updateFn func(id uint64, req dto.UpdateProductRequest) (*dto.ProductResponse, error)
}

func (m *mockProductService) Create(_ context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error) {
	return m.createFn(req)
}

func (m *mockProductService) Update(_ context.Context, id uint64, req dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	return m.updateFn(id, req)
}

func (m *mockProductService) GetByID(_ context.Context, id uint64) (*dto.ProductResponse, error) {
	return m.getFn(id)
}

func (m *mockProductService) List(_ context.Context, query dto.ProductQuery) (*dto.ProductListResponse, error) {
	return m.listFn(query)
}

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
