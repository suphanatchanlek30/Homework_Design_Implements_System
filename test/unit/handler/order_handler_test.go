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

// Confirm returns the mocked order-confirm result used by handler tests.
// คืนผล confirm order แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockOrderService) Confirm(_ context.Context, idempotencyKey string, req dto.OrderConfirmRequest) (*dto.OrderConfirmResponse, error) {
	return m.confirmFn(idempotencyKey, req)
}

// List returns the mocked order-list result used by handler tests.
// คืนผล list orders แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockOrderService) List(_ context.Context, query dto.OrderListQuery) (*dto.OrderListResponse, error) {
	return m.listFn(query)
}

// GetByID returns the mocked order-detail result used by handler tests.
// คืนผล order detail แบบ mock สำหรับใช้ใน unit test ของ handler
func (m *mockOrderService) GetByID(_ context.Context, id uint64, requesterUserID *uint64) (*dto.OrderDetailResponse, error) {
	return m.getFn(id, requesterUserID)
}

// TestOrderConfirm_Success verifies order confirmation returns HTTP 201 on success.
// ตรวจว่า order confirmation ที่สำเร็จจะตอบกลับ HTTP 201
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

// TestOrderConfirm_MissingIdempotencyKey verifies missing idempotency keys return HTTP 400.
// ตรวจว่าถ้าไม่ส่ง idempotency key ระบบจะตอบกลับ HTTP 400
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

// TestOrderConfirm_PriceChanged verifies price mismatches return HTTP 409.
// ตรวจว่ากรณีราคาที่ผู้ใช้ยอมรับไม่ตรงจะตอบกลับ HTTP 409
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

// TestOrderList_InvalidDateRange verifies invalid date ranges return HTTP 400.
// ตรวจว่าช่วงวันที่ไม่ถูกต้องใน order list จะตอบกลับ HTTP 400
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

// TestOrderGetByID_AccessDenied verifies access-denied order detail requests return HTTP 403.
// ตรวจว่าการขอดู order detail โดยไม่มีสิทธิ์จะตอบกลับ HTTP 403
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
