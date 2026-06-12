package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
)

var (
	ErrCalculationLogNotFound    = errors.New("calculation log not found")
	ErrReplayModeNotSupported    = errors.New("replay mode not supported")
	ErrCalculationReplayFailed   = errors.New("calculation replay failed")
)

type CalculationLogService interface {
	List(ctx context.Context, query dto.CalculationLogQuery) (*dto.CalculationLogListResponse, error)
	GetByCalculationID(ctx context.Context, calculationID string) (*dto.CalculationLogDetailResponse, error)
	Replay(ctx context.Context, calculationID string, req dto.CalculationLogReplayRequest) (*dto.CalculationLogReplayResponse, error)
}

type calculationLogService struct {
	repo    repository.CalculationLogRepository
	pricing PricingService
}

type storedCalculationSnapshot struct {
	Request  dto.PricingCalculateRequest `json:"request"`
	Response dto.PricingResultResponse   `json:"response"`
	Explain  bool                        `json:"explain"`
}

// NewCalculationLogService exposes list, detail, and replay operations over persisted pricing logs.
// เปิดความสามารถสำหรับ list, detail และ replay บน pricing logs ที่บันทึกไว้
func NewCalculationLogService(repo repository.CalculationLogRepository, pricing PricingService) CalculationLogService {
	return &calculationLogService{
		repo:    repo,
		pricing: pricing,
	}
}

// List returns paginated calculation logs and derives summary counts from each stored snapshot.
// คืน calculation logs แบบแบ่งหน้าและสรุปจำนวนโปรที่ถูกใช้หรือถูกข้ามจาก snapshot
func (s *calculationLogService) List(ctx context.Context, query dto.CalculationLogQuery) (*dto.CalculationLogListResponse, error) {
	page := normalizePage(query.Page)
	limit := normalizeLimit(query.Limit)

	logs, total, err := s.repo.List(ctx, repository.CalculationLogFilter{
		RequestID:   query.RequestID,
		OrderID:     query.OrderID,
		UserID:      query.UserID,
		PromotionID: query.PromotionID,
		CreatedFrom: query.CreatedFrom,
		CreatedTo:   query.CreatedTo,
	}, page, limit, query.Sort)
	if err != nil {
		return nil, err
	}

	items := make([]dto.CalculationLogSummaryResponse, len(logs))
	for i, logRow := range logs {
		summary, err := s.summaryFromLog(&logRow)
		if err != nil {
			return nil, err
		}
		items[i] = *summary
	}

	return &dto.CalculationLogListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: calcTotalPages(total, limit),
		},
	}, nil
}

// GetByCalculationID expands a single calculation log into its full audit payload.
// แตก calculation log หนึ่งรายการออกเป็นข้อมูล audit ฉบับเต็ม
func (s *calculationLogService) GetByCalculationID(ctx context.Context, calculationID string) (*dto.CalculationLogDetailResponse, error) {
	logRow, err := s.repo.FindByCalculationID(ctx, calculationID)
	if err != nil {
		return nil, ErrCalculationLogNotFound
	}

	summary, err := s.summaryFromLog(logRow)
	if err != nil {
		return nil, err
	}
	snapshot, stored, err := decodeStoredCalculationSnapshot(logRow.CalculationSnapshotJSON)
	if err != nil {
		return nil, ErrCalculationReplayFailed
	}

	return &dto.CalculationLogDetailResponse{
		CalculationLogSummaryResponse: *summary,
		AppliedPromotions:             stored.Response.AppliedPromotions,
		SkippedPromotions:             stored.Response.SkippedPromotions,
		CalculationSnapshot:           snapshot,
	}, nil
}

// Replay reruns the original request in preview mode and compares the new result against the stored snapshot.
// นำ request เดิมมารันใหม่แบบ preview แล้วเทียบผลกับ snapshot ที่เก็บไว้
func (s *calculationLogService) Replay(ctx context.Context, calculationID string, req dto.CalculationLogReplayRequest) (*dto.CalculationLogReplayResponse, error) {
	mode := req.Mode
	if mode == "" {
		mode = "SNAPSHOT_CONFIG"
	}
	if mode != "SNAPSHOT_CONFIG" {
		return nil, ErrReplayModeNotSupported
	}

	logRow, err := s.repo.FindByCalculationID(ctx, calculationID)
	if err != nil {
		return nil, ErrCalculationLogNotFound
	}

	_, stored, err := decodeStoredCalculationSnapshot(logRow.CalculationSnapshotJSON)
	if err != nil {
		return nil, ErrCalculationReplayFailed
	}

	replayResult, err := s.pricing.Preview(ctx, stored.Request)
	if err != nil {
		return nil, ErrCalculationReplayFailed
	}

	originalResult := stored.Response
	differences := comparePricingResults(originalResult, *replayResult)

	return &dto.CalculationLogReplayResponse{
		CalculationID:  calculationID,
		Mode:          mode,
		OriginalResult: originalResult,
		ReplayResult:  *replayResult,
		Matched:       len(differences) == 0,
		Differences:   differences,
	}, nil
}

// summaryFromLog derives lightweight counters and totals from the stored snapshot blob.
// ดึงตัวเลขสรุปแบบย่อจาก snapshot ที่เก็บเป็น JSON blob
func (s *calculationLogService) summaryFromLog(logRow *model.PromotionCalculationLog) (*dto.CalculationLogSummaryResponse, error) {
	var snapshot storedCalculationSnapshot
	if err := json.Unmarshal(logRow.CalculationSnapshotJSON, &snapshot); err != nil {
		return nil, err
	}
	appliedCount := len(snapshot.Response.AppliedPromotions)
	skippedCount := len(snapshot.Response.SkippedPromotions)
	return &dto.CalculationLogSummaryResponse{
		CalculationID:         logRow.CalculationID,
		OrderID:               logRow.OrderID,
		RequestID:             logRow.RequestID,
		UserID:                logRow.UserID,
		OriginalTotal:         logRow.OriginalTotal,
		DiscountTotal:         logRow.DiscountTotal,
		FinalTotal:            logRow.FinalTotal,
		AppliedPromotionCount: appliedCount,
		SkippedPromotionCount: skippedCount,
		CreatedAt:             logRow.CreatedAt,
	}, nil
}

// decodeStoredCalculationSnapshot returns both the typed replay payload and the generic JSON body used in responses.
// แปลง snapshot ที่เก็บไว้ให้ได้ทั้งโครงสร้างแบบ typed และ map ทั่วไปสำหรับ response
func decodeStoredCalculationSnapshot(raw []byte) (map[string]any, storedCalculationSnapshot, error) {
	var snapshot storedCalculationSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, storedCalculationSnapshot{}, err
	}
	var snapshotMap map[string]any
	if err := json.Unmarshal(raw, &snapshotMap); err != nil {
		return nil, storedCalculationSnapshot{}, err
	}
	return snapshotMap, snapshot, nil
}

// comparePricingResults highlights business-level differences between the stored result and a replayed result.
// เปรียบเทียบผลคำนวณเดิมกับผล replay แล้วสรุปความต่างในระดับ business result
func comparePricingResults(original, replay dto.PricingResultResponse) []string {
	differences := make([]string, 0)
	if original.OriginalTotal != replay.OriginalTotal {
		differences = append(differences, fmt.Sprintf("originalTotal: %d != %d", original.OriginalTotal, replay.OriginalTotal))
	}
	if original.DiscountTotal != replay.DiscountTotal {
		differences = append(differences, fmt.Sprintf("discountTotal: %d != %d", original.DiscountTotal, replay.DiscountTotal))
	}
	if original.FinalTotal != replay.FinalTotal {
		differences = append(differences, fmt.Sprintf("finalTotal: %d != %d", original.FinalTotal, replay.FinalTotal))
	}
	if original.Currency != replay.Currency {
		differences = append(differences, fmt.Sprintf("currency: %s != %s", original.Currency, replay.Currency))
	}
	if !reflect.DeepEqual(normalizePricingItems(original.Items), normalizePricingItems(replay.Items)) {
		differences = append(differences, "items differ")
	}
	if !reflect.DeepEqual(original.AppliedPromotions, replay.AppliedPromotions) {
		differences = append(differences, "applied promotions differ")
	}
	if !reflect.DeepEqual(original.SkippedPromotions, replay.SkippedPromotions) {
		differences = append(differences, "skipped promotions differ")
	}
	return differences
}

// normalizePricingItems strips fields that are irrelevant to replay equality checks.
// ตัด field ที่ไม่สำคัญต่อการเทียบ replay ออกก่อนตรวจความเท่ากัน
func normalizePricingItems(items []dto.PricingItemResponse) []dto.PricingItemResponse {
	cloned := make([]dto.PricingItemResponse, len(items))
	for i, item := range items {
		cloned[i] = dto.PricingItemResponse{
			ProductID:      item.ProductID,
			SKU:            item.SKU,
			ProductName:    item.ProductName,
			Quantity:       item.Quantity,
			UnitPrice:      item.UnitPrice,
			OriginalAmount: item.OriginalAmount,
			DiscountAmount: item.DiscountAmount,
			FinalAmount:    item.FinalAmount,
		}
	}
	return cloned
}
