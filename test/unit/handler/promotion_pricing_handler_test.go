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

type mockPromotionService struct {
	createFn   func(req dto.PromotionCreateRequest) (*dto.PromotionSummaryResponse, error)
	listFn     func(query dto.PromotionListQuery) (*dto.PromotionListResponse, error)
	getFn      func(id uint64) (*dto.PromotionDetailResponse, error)
	replaceFn  func(id uint64, req dto.PromotionReplaceRequest) (*dto.PromotionDetailResponse, error)
	patchFn    func(id uint64, req dto.PromotionPatchRequest) (*dto.PromotionDetailResponse, error)
	validateFn func(id uint64, req dto.PromotionValidateRequest) (*dto.PromotionValidationResponse, error)
	activateFn func(id uint64, req dto.PromotionActivateRequest) (*dto.PromotionSummaryResponse, error)
	deactivateFn func(id uint64, req dto.PromotionDeactivateRequest) (*dto.PromotionSummaryResponse, error)
	usagesFn   func(id uint64, query dto.PromotionUsageQuery) (*dto.PromotionUsageResponse, error)
}

// Create returns the mocked promotion-create result used by handler tests.
// คืนผลสร้าง promotion แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPromotionService) Create(_ context.Context, req dto.PromotionCreateRequest) (*dto.PromotionSummaryResponse, error) {
	return m.createFn(req)
}
// List returns the mocked promotion-list result used by handler tests.
// คืนผล list promotions แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPromotionService) List(_ context.Context, query dto.PromotionListQuery) (*dto.PromotionListResponse, error) {
	return m.listFn(query)
}
// GetByID returns the mocked promotion-detail result used by handler tests.
// คืนผล promotion detail แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPromotionService) GetByID(_ context.Context, id uint64) (*dto.PromotionDetailResponse, error) {
	return m.getFn(id)
}
// Replace returns the mocked promotion-replace result used by handler tests.
// คืนผล replace promotion แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPromotionService) Replace(_ context.Context, id uint64, req dto.PromotionReplaceRequest) (*dto.PromotionDetailResponse, error) {
	return m.replaceFn(id, req)
}
// Patch returns the mocked promotion-patch result used by handler tests.
// คืนผล patch promotion แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPromotionService) Patch(_ context.Context, id uint64, req dto.PromotionPatchRequest) (*dto.PromotionDetailResponse, error) {
	return m.patchFn(id, req)
}
// Validate returns the mocked promotion-validation result used by handler tests.
// คืนผล validate promotion แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPromotionService) Validate(_ context.Context, id uint64, req dto.PromotionValidateRequest) (*dto.PromotionValidationResponse, error) {
	return m.validateFn(id, req)
}
// Activate returns the mocked promotion-activate result used by handler tests.
// คืนผล activate promotion แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPromotionService) Activate(_ context.Context, id uint64, req dto.PromotionActivateRequest) (*dto.PromotionSummaryResponse, error) {
	return m.activateFn(id, req)
}
// Deactivate returns the mocked promotion-deactivate result used by handler tests.
// คืนผล deactivate promotion แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPromotionService) Deactivate(_ context.Context, id uint64, req dto.PromotionDeactivateRequest) (*dto.PromotionSummaryResponse, error) {
	return m.deactivateFn(id, req)
}
// Usages returns the mocked promotion-usage result used by handler tests.
// คืนผล promotion usage แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPromotionService) Usages(_ context.Context, id uint64, query dto.PromotionUsageQuery) (*dto.PromotionUsageResponse, error) {
	return m.usagesFn(id, query)
}

type mockPricingService struct {
	calculateFn func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
	explainFn   func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
	previewFn   func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
}

// Calculate returns the mocked pricing-calculate result used by handler tests.
// คืนผล pricing calculate แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPricingService) Calculate(_ context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return m.calculateFn(req)
}

// Explain returns the mocked pricing-explain result used by handler tests.
// คืนผล pricing explain แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPricingService) Explain(_ context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return m.explainFn(req)
}

// Preview returns the mocked pricing-preview result used by handler tests.
// คืนผล pricing preview แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockPricingService) Preview(_ context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return m.previewFn(req)
}

// TestPromotionCreate_AcceptsValidPayload verifies valid promotion payloads return HTTP 201.
// ตรวจว่า payload promotion ที่ถูกต้องจะตอบกลับ HTTP 201
func TestPromotionCreate_AcceptsValidPayload(t *testing.T) {
	app := fiber.New()
	svc := &mockPromotionService{
		createFn: func(req dto.PromotionCreateRequest) (*dto.PromotionSummaryResponse, error) {
			return &dto.PromotionSummaryResponse{PromotionID: 1, Code: req.Code, Status: "DRAFT", Version: 1}, nil
		},
	}

	app.Post("/api/v1/promotions", handler.NewPromotionHandler(svc).Create)

	body := `{
		"code":"ITEM1_10_PERCENT",
		"name":"Item 1 Discount",
		"scope":"ITEM",
		"priority":10,
		"stackable":true,
		"exclusive":false,
		"stopProcessing":false,
		"startsAt":"2026-01-01T00:00:00Z",
		"endsAt":"2026-12-31T23:59:59Z",
		"targets":[{"targetType":"PRODUCT","targetId":1}],
		"actions":[{"actionType":"PERCENTAGE_DISCOUNT","valueBasisPoints":1000,"appliesTo":"ITEM"}]
	}`

	req := httptest.NewRequest("POST", "/api/v1/promotions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

// TestPromotionCreate_CodeConflict verifies duplicate promotion codes return HTTP 409.
// ตรวจว่า promotion code ซ้ำจะตอบกลับ HTTP 409
func TestPromotionCreate_CodeConflict(t *testing.T) {
	app := fiber.New()
	svc := &mockPromotionService{
		createFn: func(req dto.PromotionCreateRequest) (*dto.PromotionSummaryResponse, error) {
			return nil, service.ErrPromotionCodeAlreadyExists
		},
	}

	app.Post("/api/v1/promotions", handler.NewPromotionHandler(svc).Create)

	req := httptest.NewRequest("POST", "/api/v1/promotions", strings.NewReader(`{"code":"DUP","name":"A","scope":"ITEM","priority":0,"startsAt":"2026-01-01T00:00:00Z","endsAt":"2026-12-31T23:59:59Z","targets":[{"targetType":"PRODUCT","targetId":1}],"actions":[{"actionType":"PERCENTAGE_DISCOUNT","valueBasisPoints":1000,"appliesTo":"ITEM"}]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "PROMOTION_CODE_ALREADY_EXISTS", body.Error.Code)
}

// TestPromotionList_InvalidStatusQuery verifies invalid promotion status filters return HTTP 400.
// ตรวจว่า status filter ของ promotion ที่ไม่ถูกต้องจะตอบกลับ HTTP 400
func TestPromotionList_InvalidStatusQuery(t *testing.T) {
	app := fiber.New()
	svc := &mockPromotionService{
		listFn: func(query dto.PromotionListQuery) (*dto.PromotionListResponse, error) {
			return &dto.PromotionListResponse{}, nil
		},
	}

	app.Get("/api/v1/promotions", handler.NewPromotionHandler(svc).List)

	req := httptest.NewRequest("GET", "/api/v1/promotions?status=BAD", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// TestPricingCalculate_EmptyItems verifies empty pricing items return HTTP 422.
// ตรวจว่า pricing request ที่ไม่มี items จะตอบกลับ HTTP 422
func TestPricingCalculate_EmptyItems(t *testing.T) {
	app := fiber.New()
	svc := &mockPricingService{
		calculateFn: func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
			return nil, service.ErrEmptyOrderItems
		},
	}

	app.Post("/api/v1/pricing/calculate", handler.NewPricingHandler(svc).Calculate)

	req := httptest.NewRequest("POST", "/api/v1/pricing/calculate", strings.NewReader(`{"items":[],"currency":"THB"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "EMPTY_ORDER_ITEMS", body.Error.Code)
}

// TestPricingExplain_Success verifies the explain endpoint returns HTTP 200 on success.
// ตรวจว่า endpoint pricing explain ที่สำเร็จจะตอบกลับ HTTP 200
func TestPricingExplain_Success(t *testing.T) {
	app := fiber.New()
	svc := &mockPricingService{
		explainFn: func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
			return &dto.PricingResultResponse{
				CalculationID: "calc-1",
				OriginalTotal: 100000,
				DiscountTotal: 10000,
				FinalTotal:    90000,
				Currency:      "THB",
				Items:         []dto.PricingItemResponse{},
				AppliedPromotions: []dto.PricingPromotionAppliedResponse{},
				SkippedPromotions: []dto.PricingPromotionSkippedResponse{},
			}, nil
		},
	}

	app.Post("/api/v1/pricing/explain", handler.NewPricingHandler(svc).Explain)

	req := httptest.NewRequest("POST", "/api/v1/pricing/explain", strings.NewReader(`{"items":[{"productId":1,"quantity":1}],"currency":"THB"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// TestPromotionActivate_VersionConflict verifies activation version conflicts return HTTP 409.
// ตรวจว่า activation ที่ version ไม่ตรงจะตอบกลับ HTTP 409
func TestPromotionActivate_VersionConflict(t *testing.T) {
	app := fiber.New()
	svc := &mockPromotionService{
		activateFn: func(id uint64, req dto.PromotionActivateRequest) (*dto.PromotionSummaryResponse, error) {
			return nil, service.ErrPromotionVersionConflict
		},
	}

	app.Post("/api/v1/promotions/:promotionId/activate", handler.NewPromotionHandler(svc).Activate)

	req := httptest.NewRequest("POST", "/api/v1/promotions/1/activate", strings.NewReader(`{"expectedVersion":1}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

// TestPromotionUsageQuery_InvalidDate verifies invalid usage-date filters return HTTP 400.
// ตรวจว่า filter วันที่ของ promotion usage ที่ไม่ถูกต้องจะตอบกลับ HTTP 400
func TestPromotionUsageQuery_InvalidDate(t *testing.T) {
	app := fiber.New()
	svc := &mockPromotionService{
		usagesFn: func(id uint64, query dto.PromotionUsageQuery) (*dto.PromotionUsageResponse, error) {
			return &dto.PromotionUsageResponse{}, nil
		},
	}

	app.Get("/api/v1/promotions/:promotionId/usages", handler.NewPromotionHandler(svc).Usages)

	req := httptest.NewRequest("GET", "/api/v1/promotions/1/usages?from=bad-date", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// TestPromotionList_AcceptsTimeFilter verifies the list endpoint accepts a valid activeAt filter.
// ตรวจว่า endpoint list promotions รับ activeAt ที่ถูกต้องและตอบกลับได้ตามปกติ
func TestPromotionList_AcceptsTimeFilter(t *testing.T) {
	app := fiber.New()
	svc := &mockPromotionService{
		listFn: func(query dto.PromotionListQuery) (*dto.PromotionListResponse, error) {
			return &dto.PromotionListResponse{
				Items: []dto.PromotionSummaryResponse{},
				Pagination: dto.Pagination{Page: 1, Limit: 10, TotalItems: 0, TotalPages: 0},
			}, nil
		},
	}

	app.Get("/api/v1/promotions", handler.NewPromotionHandler(svc).List)

	req := httptest.NewRequest("GET", "/api/v1/promotions?activeAt=2026-06-10T00:00:00Z", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
