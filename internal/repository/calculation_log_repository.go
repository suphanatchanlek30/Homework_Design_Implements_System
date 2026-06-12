package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"gorm.io/gorm"
)

type CalculationLogFilter struct {
	RequestID   *string
	OrderID     *uint64
	UserID      *uint64
	PromotionID *uint64
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type CalculationLogRepository interface {
	List(ctx context.Context, filter CalculationLogFilter, page, limit int, sort *string) ([]model.PromotionCalculationLog, int64, error)
	FindByCalculationID(ctx context.Context, calculationID string) (*model.PromotionCalculationLog, error)
}

type calculationLogRepository struct {
	db *gorm.DB
}

// NewCalculationLogRepository builds the repository used to browse and replay stored pricing logs.
// สร้าง repository สำหรับค้นหาและ replay pricing logs ที่บันทึกไว้
func NewCalculationLogRepository(db *gorm.DB) CalculationLogRepository {
	return &calculationLogRepository{db: db}
}

// List returns paginated calculation logs filtered by request, order, user, promotion, and time.
// คืน calculation logs แบบแบ่งหน้าตาม request, order, user, promotion และช่วงเวลา
func (r *calculationLogRepository) List(ctx context.Context, filter CalculationLogFilter, page, limit int, sort *string) ([]model.PromotionCalculationLog, int64, error) {
	var logs []model.PromotionCalculationLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.PromotionCalculationLog{})
	if filter.RequestID != nil && *filter.RequestID != "" {
		query = query.Where("request_id = ?", *filter.RequestID)
	}
	if filter.OrderID != nil {
		query = query.Where("order_id = ?", *filter.OrderID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}
	if filter.PromotionID != nil {
		query = query.Where("JSON_CONTAINS(applied_promotions_json, JSON_OBJECT('promotionId', CAST(? AS UNSIGNED)), '$')", *filter.PromotionID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if sort != nil && *sort != "" {
		query = query.Order(*sort)
	} else {
		query = query.Order("created_at DESC, id DESC")
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindByCalculationID loads one calculation log by its external calculation identifier.
// โหลด calculation log หนึ่งรายการจาก calculation ID ที่ใช้ภายนอก
func (r *calculationLogRepository) FindByCalculationID(ctx context.Context, calculationID string) (*model.PromotionCalculationLog, error) {
	var logRow model.PromotionCalculationLog
	if err := r.db.WithContext(ctx).Where("calculation_id = ?", calculationID).First(&logRow).Error; err != nil {
		return nil, err
	}
	return &logRow, nil
}

// calculationLogPromotionFilterClause builds the JSON filter expression for one promotion ID.
// สร้างเงื่อนไข JSON สำหรับกรอง calculation log ตาม promotion ID
func calculationLogPromotionFilterClause(promotionID uint64) string {
	return fmt.Sprintf("JSON_CONTAINS(applied_promotions_json, JSON_OBJECT('promotionId', CAST(%d AS UNSIGNED)), '$')", promotionID)
}
