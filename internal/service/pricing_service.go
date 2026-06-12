package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/promotion"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrEmptyOrderItems   = errors.New("empty order items")
	ErrInvalidQuantity   = errors.New("invalid quantity")
	ErrProductInactive   = errors.New("product inactive")
	ErrCurrencyMismatch  = errors.New("currency mismatch")
	ErrCalculationFailed = errors.New("calculation failed")
)

type PricingService interface {
	Calculate(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
	Explain(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
	Preview(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error)
}

type pricingService struct {
	db            *gorm.DB
	productRepo   repository.ProductRepository
	promotionRepo repository.PromotionRepository
	calculator    promotion.Calculator
}

// NewPricingService wires product loading, promotion loading, and the promotion engine into one pricing use case.
// ประกอบ dependency ที่ใช้โหลดสินค้า โหลดโปร และเรียก promotion engine เข้าด้วยกัน
func NewPricingService(db *gorm.DB, productRepo repository.ProductRepository, promotionRepo repository.PromotionRepository) PricingService {
	return &pricingService{
		db:            db,
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		calculator:    promotion.NewCalculator(),
	}
}

// Calculate returns the final totals and persists an audit log for later inspection.
// คำนวณราคาสุดท้ายและบันทึก audit log ไว้สำหรับตรวจย้อนหลัง
func (s *pricingService) Calculate(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return s.calculate(ctx, req, false, true)
}

// Explain runs the same pricing flow as Calculate but marks the saved snapshot as an explain request.
// ใช้ flow คำนวณราคาเดียวกับ Calculate แต่ติดธงว่าเป็นคำขอแบบ explain
func (s *pricingService) Explain(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return s.calculate(ctx, req, true, true)
}

// Preview recalculates pricing without writing a new calculation log entry.
// คำนวณราคาใหม่โดยไม่สร้าง calculation log เพิ่ม ใช้กับ replay หรือ preview
func (s *pricingService) Preview(ctx context.Context, req dto.PricingCalculateRequest) (*dto.PricingResultResponse, error) {
	return s.calculate(ctx, req, false, false)
}

// calculate is the shared orchestration path that validates inputs, loads dependencies, runs the engine, and maps the response.
// เป็น flow กลางที่ตรวจ input โหลดข้อมูลที่ต้องใช้ เรียก engine และแปลงผลตอบกลับ
func (s *pricingService) calculate(ctx context.Context, req dto.PricingCalculateRequest, explain bool, persistLog bool) (*dto.PricingResultResponse, error) {
	if len(req.Items) == 0 {
		return nil, ErrEmptyOrderItems
	}

	aggregated, err := aggregateItems(req.Items)
	if err != nil {
		return nil, err
	}

	products, err := s.productRepo.FindByIDs(ctx, mapKeys(aggregated))
	if err != nil {
		return nil, err
	}
	if len(products) != len(aggregated) {
		return nil, ErrProductNotFound
	}

	productMap := make(map[uint64]model.Product, len(products))
	for _, product := range products {
		if product.Status != "ACTIVE" {
			return nil, ErrProductInactive
		}
		productMap[product.ID] = product
	}

	currency := req.Currency
	if currency == "" {
		currency = "THB"
	}
	if currency != "THB" {
		return nil, ErrCurrencyMismatch
	}

	items := make([]promotion.CalculationItem, 0, len(aggregated))
	keys := mapKeys(aggregated)
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, id := range keys {
		quantity := aggregated[id]
		product := productMap[id]
		if product.Currency != currency {
			return nil, ErrCurrencyMismatch
		}
		items = append(items, promotion.CalculationItem{
			ProductID:   product.ID,
			SKU:         product.SKU,
			ProductName: product.Name,
			CategoryID:  product.CategoryID,
			Quantity:    quantity,
			UnitPrice:   product.PriceAmount,
		})
	}

	promotions, err := s.promotionRepo.FindActivePromotions(ctx, time.Now())
	if err != nil {
		return nil, err
	}

	result, err := s.calculator.Calculate(ctx, promotion.CalculationContext{
		Now:            time.Now(),
		UserID:         req.UserID,
		Currency:       currency,
		CouponCodes:    req.CouponCodes,
		PaymentMethod:  req.PaymentMethod,
		ShippingMethod: shippingMethod(req.Shipping),
		Items:          items,
		Promotions:     promotions,
	})
	if err != nil {
		return nil, ErrCalculationFailed
	}

	response := &dto.PricingResultResponse{
		CalculationID:     result.CalculationID,
		OriginalTotal:     result.OriginalTotal,
		DiscountTotal:     result.DiscountTotal,
		FinalTotal:        result.FinalTotal,
		Currency:          result.Currency,
		Items:             make([]dto.PricingItemResponse, len(result.Items)),
		AppliedPromotions: make([]dto.PricingPromotionAppliedResponse, len(result.AppliedPromotions)),
		SkippedPromotions: make([]dto.PricingPromotionSkippedResponse, len(result.SkippedPromotions)),
	}

	for i, item := range result.Items {
		response.Items[i] = dto.PricingItemResponse{
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

	for i, applied := range result.AppliedPromotions {
		response.AppliedPromotions[i] = dto.PricingPromotionAppliedResponse{
			PromotionID:    applied.PromotionID,
			Code:           applied.Code,
			Name:           applied.Name,
			Scope:          applied.Scope,
			DiscountAmount: applied.DiscountAmount,
		}
	}

	for i, skipped := range result.SkippedPromotions {
		response.SkippedPromotions[i] = dto.PricingPromotionSkippedResponse{
			PromotionID: skipped.PromotionID,
			Code:        skipped.Code,
			Name:        skipped.Name,
			Reason:      skipped.Reason,
		}
	}

	if persistLog {
		if err := s.persistCalculationLog(ctx, result, req, response, explain); err != nil {
			return nil, err
		}
	}

	return response, nil
}

// persistCalculationLog stores enough request and response context to debug or replay a pricing result later.
// เก็บ request/response และข้อมูลประกอบให้พอสำหรับ debug หรือ replay ผลคำนวณภายหลัง
func (s *pricingService) persistCalculationLog(ctx context.Context, result *promotion.CalculationResult, req dto.PricingCalculateRequest, response *dto.PricingResultResponse, explain bool) error {
	appliedJSON, _ := json.Marshal(response.AppliedPromotions)
	skippedJSON, _ := json.Marshal(response.SkippedPromotions)
	snapshotJSON, _ := json.Marshal(map[string]any{
		"request":       req,
		"response":      response,
		"explain":       explain,
		"decisionTrace": result.DecisionTrace,
		"scopeOrder":    result.Snapshot["scopeOrder"],
	})

	logRow := model.PromotionCalculationLog{
		CalculationID:           result.CalculationID,
		RequestID:               fmt.Sprintf("req-%d", time.Now().UnixNano()),
		UserID:                  req.UserID,
		OriginalTotal:           result.OriginalTotal,
		DiscountTotal:           result.DiscountTotal,
		FinalTotal:              result.FinalTotal,
		AppliedPromotionsJSON:   appliedJSON,
		SkippedPromotionsJSON:   skippedJSON,
		CalculationSnapshotJSON: snapshotJSON,
	}

	return s.db.WithContext(ctx).Create(&logRow).Error
}

// aggregateItems merges duplicate product IDs and rejects zero or negative quantities up front.
// รวมรายการสินค้าที่ product ซ้ำกันและกัน quantity ที่ไม่ถูกต้องตั้งแต่ต้นทาง
func aggregateItems(items []dto.PricingItemRequest) (map[uint64]int, error) {
	aggregated := make(map[uint64]int)
	for _, item := range items {
		if item.Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}
		if _, exists := aggregated[item.ProductID]; exists {
			aggregated[item.ProductID] += item.Quantity
		} else {
			aggregated[item.ProductID] = item.Quantity
		}
	}
	return aggregated, nil
}

// mapKeys flattens the aggregated product map into a list usable for repository lookups and deterministic sorting.
// ดึง product IDs ออกจาก map เพื่อใช้ query repository และจัดลำดับแบบคงที่
func mapKeys(items map[uint64]int) []uint64 {
	keys := make([]uint64, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	return keys
}

// shippingMethod converts the optional shipping DTO into the pointer shape expected by the engine context.
// แปลง shipping DTO ที่อาจไม่มีค่าให้เป็นรูปแบบ pointer ที่ engine ใช้งาน
func shippingMethod(value *dto.PricingShippingRequest) *string {
	if value == nil {
		return nil
	}
	copyValue := value.Method
	return &copyValue
}
