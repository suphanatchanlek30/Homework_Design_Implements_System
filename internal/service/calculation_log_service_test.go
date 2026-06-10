package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
)

type mockCalculationLogRepo struct {
	findFn func(calculationID string) (*model.PromotionCalculationLog, error)
	listFn func(filter repository.CalculationLogFilter, page, limit int, sort *string) ([]model.PromotionCalculationLog, int64, error)
}

func (m *mockCalculationLogRepo) List(ctx context.Context, filter repository.CalculationLogFilter, page, limit int, sort *string) ([]model.PromotionCalculationLog, int64, error) {
	return m.listFn(filter, page, limit, sort)
}

func (m *mockCalculationLogRepo) FindByCalculationID(ctx context.Context, calculationID string) (*model.PromotionCalculationLog, error) {
	return m.findFn(calculationID)
}

type mockPreviewPricingService struct {
	previewFn func(req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
}

func (m *mockPreviewPricingService) Calculate(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return nil, nil
}

func (m *mockPreviewPricingService) Explain(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return nil, nil
}

func (m *mockPreviewPricingService) Preview(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return m.previewFn(req)
}

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

	svc := NewCalculationLogService(repo, pricing)
	res, err := svc.Replay(context.Background(), "calc-1", dto.CalculationLogReplayRequest{Mode: "SNAPSHOT_CONFIG"})

	assert.NoError(t, err)
	assert.True(t, res.Matched)
	assert.Empty(t, res.Differences)
	assert.Equal(t, "calc-1", res.CalculationID)
}
