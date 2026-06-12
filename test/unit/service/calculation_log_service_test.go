package service_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
	servicepkg "github.com/suphanatchanlek30/homework_design_implements_system/internal/service"
)

type mockCalculationLogRepo struct {
	findFn func(calculationID string) (*model.PromotionCalculationLog, error)
	listFn func(filter repository.CalculationLogFilter, page, limit int, sort *string) ([]model.PromotionCalculationLog, int64, error)
}

// List returns the mocked paginated calculation logs for service tests.
// คืน calculation logs แบบ mock สำหรับใช้ใน unit test ของ service
func (m *mockCalculationLogRepo) List(ctx context.Context, filter repository.CalculationLogFilter, page, limit int, sort *string) ([]model.PromotionCalculationLog, int64, error) {
	return m.listFn(filter, page, limit, sort)
}

// FindByCalculationID returns the mocked calculation log chosen by the test case.
// คืน calculation log แบบ mock ตาม calculation ID ที่ test กำหนดไว้
func (m *mockCalculationLogRepo) FindByCalculationID(ctx context.Context, calculationID string) (*model.PromotionCalculationLog, error) {
	return m.findFn(calculationID)
}

type mockPreviewPricingService struct {
	previewFn func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
}

// Calculate is unused in this test double because replay exercises preview mode only.
// เมทอดนี้ไม่ถูกใช้ใน test double นี้เพราะการ replay ใช้เฉพาะ preview mode
func (m *mockPreviewPricingService) Calculate(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return nil, nil
}

// Explain is unused in this test double because replay exercises preview mode only.
// เมทอดนี้ไม่ถูกใช้ใน test double นี้เพราะการ replay ใช้เฉพาะ preview mode
func (m *mockPreviewPricingService) Explain(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return nil, nil
}

// Preview returns the mocked pricing result used by replay assertions.
// คืนผล pricing แบบ mock ที่ใช้ตรวจผลลัพธ์ของ replay
func (m *mockPreviewPricingService) Preview(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return m.previewFn(req)
}

// TestCalculationLogService_ReplayMatched verifies replay success when the recalculated result matches the snapshot.
// ตรวจว่า replay สำเร็จและถือว่า match เมื่อผลคำนวณใหม่ตรงกับ snapshot เดิม
func TestCalculationLogService_ReplayMatched(t *testing.T) {
	request := dto.PricingCalculateRequest{
		Currency: "THB",
		Items: []dto.PricingItemRequest{{ProductID: 1, Quantity: 1}},
	}
	original := dto.PricingResultResponse{
		CalculationID: "calc-1",
		OriginalTotal: 100000,
		DiscountTotal: 10000,
		FinalTotal:    90000,
		Currency:      "THB",
		Items: []dto.PricingItemResponse{{
			ProductID:      1,
			SKU:            "SKU-1",
			ProductName:    "Product 1",
			Quantity:       1,
			UnitPrice:      100000,
			OriginalAmount: 100000,
			DiscountAmount: 10000,
			FinalAmount:    90000,
		}},
	}
	snapshotJSON, _ := json.Marshal(map[string]any{
		"request": request,
		"response": original,
		"explain": false,
		"decisionTrace": []string{"step-1"},
	})

	repo := &mockCalculationLogRepo{
		findFn: func(calculationID string) (*model.PromotionCalculationLog, error) {
			return &model.PromotionCalculationLog{
				CalculationID:          calculationID,
				RequestID:              "req-1",
				OriginalTotal:          100000,
				DiscountTotal:          10000,
				FinalTotal:             90000,
				CalculationSnapshotJSON: snapshotJSON,
				CreatedAt:               time.Now(),
			}, nil
		},
		listFn: func(filter repository.CalculationLogFilter, page, limit int, sort *string) ([]model.PromotionCalculationLog, int64, error) {
			return nil, 0, nil
		},
	}
	pricing := &mockPreviewPricingService{
		previewFn: func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
			return &original, nil
		},
	}

	svc := servicepkg.NewCalculationLogService(repo, pricing)
	res, err := svc.Replay(context.Background(), "calc-1", dto.CalculationLogReplayRequest{Mode: "SNAPSHOT_CONFIG"})

	assert.NoError(t, err)
	assert.True(t, res.Matched)
	assert.Empty(t, res.Differences)
	assert.Equal(t, "calc-1", res.CalculationID)
}
