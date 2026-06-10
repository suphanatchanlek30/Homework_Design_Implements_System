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

type mockOrderService struct {
	confirmFn func(idempotencyKey string, req dto.OrderConfirmRequest) (*dto.OrderConfirmResponse, error)
	listFn    func(query dto.OrderListQuery) (*dto.OrderListResponse, error)
	getFn     func(id uint64, requesterUserID *uint64) (*dto.OrderDetailResponse, error)
}

func (m *mockOrderService) Confirm(_ context.Context, idempotencyKey string, req dto.OrderConfirmRequest) (*dto.OrderConfirmResponse, error) {
	return m.confirmFn(idempotencyKey, req)
}

func (m *mockOrderService) List(_ context.Context, query dto.OrderListQuery) (*dto.OrderListResponse, error) {
	return m.listFn(query)
}

func (m *mockOrderService) GetByID(_ context.Context, id uint64, requesterUserID *uint64) (*dto.OrderDetailResponse, error) {
	return m.getFn(id, requesterUserID)
}

func TestOrderConfirm_Success(t *testing.T) {
	app := fiber.New()
	svc := &mockOrderService{
		confirmFn: func(idempotencyKey string, req dto.OrderConfirmRequest) (*dto.OrderConfirmResponse, error) {
			return &dto.OrderConfirmResponse{
				OrderDetailResponse: dto.OrderDetailResponse{
					OrderSummaryResponse: dto.OrderSummaryResponse{
						OrderID:       1,
						OrderNo:       "ORD-1",
						Status:        "CONFIRMED",
						Currency:      "THB",
						OriginalTotal: 100000,
						DiscountTotal: 10000,
						FinalTotal:    90000,
						CalculationID: req.CalculationID,
					},
				},
			}, nil
		},
	}

	app.Post("/api/v1/orders/confirm", handler.NewOrderHandler(svc).Confirm)

	req := httptest.NewRequest("POST", "/api/v1/orders/confirm", strings.NewReader(`{"calculationId":"calc-1","acceptedFinalTotal":90000,"items":[{"productId":1,"quantity":1}],"currency":"THB"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestOrderConfirm_MissingIdempotencyKey(t *testing.T) {
	app := fiber.New()
	svc := &mockOrderService{
		confirmFn: func(idempotencyKey string, req dto.OrderConfirmRequest) (*dto.OrderConfirmResponse, error) {
			return nil, nil
		},
	}

	app.Post("/api/v1/orders/confirm", handler.NewOrderHandler(svc).Confirm)

	req := httptest.NewRequest("POST", "/api/v1/orders/confirm", strings.NewReader(`{"calculationId":"calc-1","acceptedFinalTotal":90000,"items":[{"productId":1,"quantity":1}]}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "IDEMPOTENCY_KEY_REQUIRED", body.Error.Code)
}

func TestOrderConfirm_PriceChanged(t *testing.T) {
	app := fiber.New()
	svc := &mockOrderService{
		confirmFn: func(idempotencyKey string, req dto.OrderConfirmRequest) (*dto.OrderConfirmResponse, error) {
			return nil, service.ErrOrderPriceChanged
		},
	}

	app.Post("/api/v1/orders/confirm", handler.NewOrderHandler(svc).Confirm)

	req := httptest.NewRequest("POST", "/api/v1/orders/confirm", strings.NewReader(`{"calculationId":"calc-1","acceptedFinalTotal":12345,"items":[{"productId":1,"quantity":1}],"currency":"THB"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-2")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestOrderList_InvalidDateRange(t *testing.T) {
	app := fiber.New()
	svc := &mockOrderService{
		listFn: func(query dto.OrderListQuery) (*dto.OrderListResponse, error) {
			return &dto.OrderListResponse{}, nil
		},
	}

	app.Get("/api/v1/orders", handler.NewOrderHandler(svc).List)

	req := httptest.NewRequest("GET", "/api/v1/orders?createdFrom=2026-06-11T00:00:00Z&createdTo=2026-06-10T00:00:00Z", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestOrderGetByID_AccessDenied(t *testing.T) {
	app := fiber.New()
	svc := &mockOrderService{
		getFn: func(id uint64, requesterUserID *uint64) (*dto.OrderDetailResponse, error) {
			return nil, service.ErrOrderAccessDenied
		},
	}

	app.Get("/api/v1/orders/:orderId", handler.NewOrderHandler(svc).GetByID)

	req := httptest.NewRequest("GET", "/api/v1/orders/1?userId=99", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}
