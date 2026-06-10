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

func (m *mockPromotionService) Create(_ context.Context, req dto.PromotionCreateRequest) (*dto.PromotionSummaryResponse, error) {
	return m.createFn(req)
}
func (m *mockPromotionService) List(_ context.Context, query dto.PromotionListQuery) (*dto.PromotionListResponse, error) {
	return m.listFn(query)
}
func (m *mockPromotionService) GetByID(_ context.Context, id uint64) (*dto.PromotionDetailResponse, error) {
	return m.getFn(id)
}
func (m *mockPromotionService) Replace(_ context.Context, id uint64, req dto.PromotionReplaceRequest) (*dto.PromotionDetailResponse, error) {
	return m.replaceFn(id, req)
}
func (m *mockPromotionService) Patch(_ context.Context, id uint64, req dto.PromotionPatchRequest) (*dto.PromotionDetailResponse, error) {
	return m.patchFn(id, req)
}
func (m *mockPromotionService) Validate(_ context.Context, id uint64, req dto.PromotionValidateRequest) (*dto.PromotionValidationResponse, error) {
	return m.validateFn(id, req)
}
func (m *mockPromotionService) Activate(_ context.Context, id uint64, req dto.PromotionActivateRequest) (*dto.PromotionSummaryResponse, error) {
	return m.activateFn(id, req)
}
func (m *mockPromotionService) Deactivate(_ context.Context, id uint64, req dto.PromotionDeactivateRequest) (*dto.PromotionSummaryResponse, error) {
	return m.deactivateFn(id, req)
}
func (m *mockPromotionService) Usages(_ context.Context, id uint64, query dto.PromotionUsageQuery) (*dto.PromotionUsageResponse, error) {
	return m.usagesFn(id, query)
}

type mockPricingService struct {
	calculateFn func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
	explainFn   func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
	previewFn   func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
}

func (m *mockPricingService) Calculate(_ context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return m.calculateFn(req)
}

func (m *mockPricingService) Explain(_ context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return m.explainFn(req)
}

func (m *mockPricingService) Preview(_ context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return m.previewFn(req)
}

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
