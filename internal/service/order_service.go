package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrIdempotencyKeyRequired = errors.New("idempotency key required")
	ErrOrderPriceChanged      = errors.New("order price changed")
	ErrPromotionUsageLimitReached = errors.New("promotion usage limit reached")
	ErrProductUnavailable     = errors.New("product unavailable")
	ErrOrderConfirmationFailed = errors.New("order confirmation failed")
	ErrOrderNotFound          = errors.New("order not found")
	ErrOrderAccessDenied      = errors.New("order access denied")
)

type OrderService interface {
	Confirm(ctx context.Context, idempotencyKey string, req dto.OrderConfirmRequest) (*dto.OrderConfirmResponse, error)
	List(ctx context.Context, query dto.OrderListQuery) (*dto.OrderListResponse, error)
	GetByID(ctx context.Context, id uint64, requesterUserID *uint64) (*dto.OrderDetailResponse, error)
}

type orderService struct {
	db           *gorm.DB
	orderRepo    repository.OrderRepository
	promotionRepo repository.PromotionRepository
	pricing      PricingService
}

// NewOrderService wires confirmation logic on top of pricing, storage, and promotion usage checks.
// ประกอบ dependency สำหรับยืนยันคำสั่งซื้อโดยอาศัย pricing, storage และการตรวจ usage ของโปร
func NewOrderService(db *gorm.DB, orderRepo repository.OrderRepository, promotionRepo repository.PromotionRepository, pricing PricingService) OrderService {
	return &orderService{
		db:            db,
		orderRepo:     orderRepo,
		promotionRepo:  promotionRepo,
		pricing:       pricing,
	}
}

// Confirm revalidates pricing, enforces idempotency, checks usage limits, and creates the final order transactionally.
// คำนวณราคาใหม่ ตรวจ idempotency ตรวจ limit การใช้โปร และสร้าง order ภายใน transaction เดียว
func (s *orderService) Confirm(ctx context.Context, idempotencyKey string, req dto.OrderConfirmRequest) (*dto.OrderConfirmResponse, error) {
	if idempotencyKey == "" {
		return nil, ErrIdempotencyKeyRequired
	}
	if req.CalculationID == "" {
		return nil, ErrOrderConfirmationFailed
	}
	if len(req.Items) == 0 {
		return nil, ErrOrderConfirmationFailed
	}

	requestHash, err := hashOrderRequest(req)
	if err != nil {
		return nil, ErrOrderConfirmationFailed
	}

	if existing, err := s.orderRepo.FindByIdempotencyKey(ctx, idempotencyKey); err == nil && existing != nil {
		if existing.RequestHash != requestHash {
			return nil, ErrOrderConfirmationFailed
		}
		return s.orderToConfirmResponse(existing)
	}

	if _, err := s.findCalculationLog(ctx, req.CalculationID); err != nil {
		return nil, ErrOrderConfirmationFailed
	}

	pricingReq := dto.PricingCalculateRequest{
		UserID:        req.UserID,
		Items:         toPricingItems(req.Items),
		CouponCodes:   req.CouponCodes,
		PaymentMethod: req.PaymentMethod,
		Shipping:      toPricingShipping(req.Shipping),
		Currency:      req.Currency,
	}

	pricingResult, err := s.pricing.Calculate(ctx, pricingReq)
	if err != nil {
		if errors.Is(err, ErrEmptyOrderItems) || errors.Is(err, ErrInvalidQuantity) || errors.Is(err, ErrProductNotFound) || errors.Is(err, ErrProductInactive) || errors.Is(err, ErrCurrencyMismatch) {
			return nil, err
		}
		return nil, ErrOrderConfirmationFailed
	}

	if pricingResult.FinalTotal != req.AcceptedFinalTotal {
		return nil, ErrOrderPriceChanged
	}

	var confirmed *model.Order
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockAndValidatePromotionUsage(tx, pricingResult, req.UserID); err != nil {
			return err
		}

		order := &model.Order{
			OrderNo:                generateOrderNo(),
			IdempotencyKey:         idempotencyKey,
			RequestHash:            requestHash,
			CalculationID:          pricingResult.CalculationID,
			UserID:                 req.UserID,
			OriginalTotal:          pricingResult.OriginalTotal,
			DiscountTotal:          pricingResult.DiscountTotal,
			FinalTotal:             pricingResult.FinalTotal,
			Currency:               pricingResult.Currency,
			Status:                 "CONFIRMED",
			AppliedPromotionsJSON:  mustJSON(pricingResult.AppliedPromotions),
			SkippedPromotionsJSON:  mustJSON(pricingResult.SkippedPromotions),
			CalculationSnapshotJSON: mustJSON(map[string]any{
				"request":  pricingReq,
				"response": pricingResult,
			}),
		}

		if err := tx.Create(order).Error; err != nil {
			return err
		}

		items := make([]model.OrderItem, len(pricingResult.Items))
		for i, item := range pricingResult.Items {
			items[i] = model.OrderItem{
				OrderID:        order.ID,
				ProductID:      item.ProductID,
				ProductName:    item.ProductName,
				SKU:            item.SKU,
				Quantity:       item.Quantity,
				UnitPrice:      item.UnitPrice,
				OriginalAmount: item.OriginalAmount,
				DiscountAmount: item.DiscountAmount,
				FinalAmount:    item.FinalAmount,
			}
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}

		for _, applied := range pricingResult.AppliedPromotions {
			usage := model.PromotionUsage{
				PromotionID:    applied.PromotionID,
				UserID:         req.UserID,
				OrderID:        &order.ID,
				UsageCount:     1,
				DiscountAmount: applied.DiscountAmount,
			}
			if err := tx.Create(&usage).Error; err != nil {
				return err
			}
		}

		confirmed = order
		return nil
	}); err != nil {
		switch {
		case errors.Is(err, ErrPromotionUsageLimitReached):
			return nil, ErrPromotionUsageLimitReached
		default:
			return nil, ErrOrderConfirmationFailed
		}
	}

	return s.orderToConfirmResponse(confirmed)
}

// List returns paginated order summaries for backoffice or audit use cases.
// คืนรายการสรุปคำสั่งซื้อแบบแบ่งหน้าเพื่อใช้ใน backoffice หรือ audit
func (s *orderService) List(ctx context.Context, query dto.OrderListQuery) (*dto.OrderListResponse, error) {
	page := normalizePage(query.Page)
	limit := normalizeLimit(query.Limit)

	orders, total, err := s.orderRepo.List(ctx, query.Status, query.UserID, query.CreatedFrom, query.CreatedTo, page, limit, query.Sort)
	if err != nil {
		return nil, err
	}

	items := make([]dto.OrderSummaryResponse, len(orders))
	for i, order := range orders {
		items[i] = orderSummaryResponse(&order)
	}

	return &dto.OrderListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: calcTotalPages(total, limit),
		},
	}, nil
}

// GetByID loads a single order and optionally enforces that the caller owns it.
// โหลดคำสั่งซื้อหนึ่งรายการและตรวจสิทธิ์ว่าเป็นของผู้เรียกหรือไม่ถ้ามีการระบุ user
func (s *orderService) GetByID(ctx context.Context, id uint64, requesterUserID *uint64) (*dto.OrderDetailResponse, error) {
	order, err := s.orderRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if requesterUserID != nil && order.UserID != nil && *requesterUserID != *order.UserID {
		return nil, ErrOrderAccessDenied
	}
	return orderDetailResponse(order)
}

// findCalculationLog ensures the order confirmation request points at a known pricing calculation.
// ตรวจว่าการยืนยัน order อ้างถึง calculation log ที่มีอยู่จริง
func (s *orderService) findCalculationLog(ctx context.Context, calculationID string) (*model.PromotionCalculationLog, error) {
	var logRow model.PromotionCalculationLog
	if err := s.db.WithContext(ctx).Where("calculation_id = ?", calculationID).First(&logRow).Error; err != nil {
		return nil, err
	}
	return &logRow, nil
}

// lockAndValidatePromotionUsage serializes usage checks so max-usage rules stay correct under concurrent confirms.
// ล็อกและตรวจ usage ของ promotion เพื่อให้ max usage ยังถูกต้องแม้มี confirm พร้อมกันหลายคำขอ
func (s *orderService) lockAndValidatePromotionUsage(tx *gorm.DB, result *dto.PricingResultResponse, userID *uint64) error {
	for _, applied := range result.AppliedPromotions {
		var promotion model.Promotion
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Targets").Preload("Conditions").Preload("Actions").First(&promotion, applied.PromotionID).Error; err != nil {
			return err
		}

		if promotion.MaxUsage != nil {
			var totalUsage int64
			if err := tx.Model(&model.PromotionUsage{}).Where("promotion_id = ?", promotion.ID).Count(&totalUsage).Error; err != nil {
				return err
			}
			if totalUsage+1 > int64(*promotion.MaxUsage) {
				return ErrPromotionUsageLimitReached
			}
		}

		if userID != nil && promotion.MaxUsagePerUser != nil {
			var userUsage int64
			if err := tx.Model(&model.PromotionUsage{}).Where("promotion_id = ? AND user_id = ?", promotion.ID, *userID).Count(&userUsage).Error; err != nil {
				return err
			}
			if userUsage+1 > int64(*promotion.MaxUsagePerUser) {
				return ErrPromotionUsageLimitReached
			}
		}
	}
	return nil
}

// orderToConfirmResponse expands a stored order into the same detail shape returned from the confirm endpoint.
// แปลง order ที่บันทึกแล้วให้มีรูปแบบเดียวกับ response ของ endpoint confirm
func (s *orderService) orderToConfirmResponse(order *model.Order) (*dto.OrderConfirmResponse, error) {
	detail, err := orderDetailResponse(order)
	if err != nil {
		return nil, err
	}
	return &dto.OrderConfirmResponse{OrderDetailResponse: *detail}, nil
}

// orderSummaryResponse maps the persisted order row into the lightweight list response.
// แปลง order model เป็น response แบบย่อสำหรับหน้า list
func orderSummaryResponse(order *model.Order) dto.OrderSummaryResponse {
	return dto.OrderSummaryResponse{
		OrderID:       order.ID,
		OrderNo:       order.OrderNo,
		UserID:        order.UserID,
		Status:        order.Status,
		Currency:      order.Currency,
		OriginalTotal: order.OriginalTotal,
		DiscountTotal: order.DiscountTotal,
		FinalTotal:    order.FinalTotal,
		CalculationID: order.CalculationID,
		CreatedAt:     order.CreatedAt,
		UpdatedAt:     order.UpdatedAt,
	}
}

// orderDetailResponse decodes embedded promotion snapshots and joins them with stored order items.
// แตก snapshot ของ promotion และประกอบรวมกับ order items ที่เก็บไว้ให้เป็น response เต็ม
func orderDetailResponse(order *model.Order) (*dto.OrderDetailResponse, error) {
	applied := make([]dto.PricingPromotionAppliedResponse, 0)
	if len(order.AppliedPromotionsJSON) > 0 {
		_ = json.Unmarshal(order.AppliedPromotionsJSON, &applied)
	}
	skipped := make([]dto.PricingPromotionSkippedResponse, 0)
	if len(order.SkippedPromotionsJSON) > 0 {
		_ = json.Unmarshal(order.SkippedPromotionsJSON, &skipped)
	}

	var snapshot map[string]any
	if len(order.CalculationSnapshotJSON) > 0 {
		_ = json.Unmarshal(order.CalculationSnapshotJSON, &snapshot)
	}

	items := make([]dto.OrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = dto.OrderItemResponse{
			ProductID:      item.ProductID,
			SKU:            item.SKU,
			ProductName:    item.ProductName,
			Quantity:       item.Quantity,
			UnitPrice:      item.UnitPrice,
			OriginalAmount: item.OriginalAmount,
			DiscountAmount: item.DiscountAmount,
			FinalAmount:    item.FinalAmount,
			CreatedAt:      item.CreatedAt,
		}
	}

	return &dto.OrderDetailResponse{
		OrderSummaryResponse: orderSummaryResponse(order),
		Items:                items,
		AppliedPromotions:    applied,
		SkippedPromotions:    skipped,
		CalculationSnapshot:  snapshot,
	}, nil
}

// toPricingItems reuses the pricing flow by translating order items into pricing request items.
// แปลง order items ให้กลับไปอยู่ในรูปแบบ input ของ pricing flow เดิม
func toPricingItems(items []dto.OrderItemRequest) []dto.PricingItemRequest {
	res := make([]dto.PricingItemRequest, len(items))
	for i, item := range items {
		res[i] = dto.PricingItemRequest{ProductID: item.ProductID, Quantity: item.Quantity}
	}
	return res
}

// toPricingShipping reuses the pricing flow by translating the optional shipping payload.
// แปลงข้อมูล shipping ของ order ให้ใช้ซ้ำกับ pricing flow ได้
func toPricingShipping(shipping *dto.OrderShippingRequest) *dto.PricingShippingRequest {
	if shipping == nil {
		return nil
	}
	return &dto.PricingShippingRequest{Method: shipping.Method}
}

// hashOrderRequest creates a stable signature used to validate repeated idempotent requests.
// สร้างลายเซ็นคงที่ของ request เพื่อเทียบคำขอ idempotent ที่ส่งซ้ำเข้ามา
func hashOrderRequest(req dto.OrderConfirmRequest) (string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// mustJSON is a small helper for embedding response snapshots into order records.
// helper เล็ก ๆ สำหรับ marshal ข้อมูล snapshot ลงใน record ของ order
func mustJSON(value any) []byte {
	raw, _ := json.Marshal(value)
	return raw
}

// generateOrderNo creates a unique order number using the current timestamp.
// สร้างเลข order ที่ไม่ซ้ำโดยอิงเวลาปัจจุบัน
func generateOrderNo() string {
	return fmt.Sprintf("ORD-%d", time.Now().UnixNano())
}
