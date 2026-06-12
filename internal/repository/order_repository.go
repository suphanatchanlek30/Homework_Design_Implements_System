package repository

import (
	"context"
	"time"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	FindByID(ctx context.Context, id uint64) (*model.Order, error)
	FindByOrderNo(ctx context.Context, orderNo string) (*model.Order, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*model.Order, error)
	List(ctx context.Context, status *string, userID *uint64, createdFrom, createdTo *time.Time, page, limit int, sort *string) ([]model.Order, int64, error)
	Update(ctx context.Context, order *model.Order) error
}

type orderRepository struct {
	*BaseRepository[model.Order]
}

// NewOrderRepository builds the order repository with the shared generic CRUD base.
// สร้าง order repository โดยใช้ generic CRUD base ร่วมกัน
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{
		BaseRepository: NewBaseRepository[model.Order](db),
	}
}

// FindByID loads one order together with its stored order items.
// โหลดคำสั่งซื้อหนึ่งรายการพร้อม order items ที่ผูกอยู่
func (r *orderRepository) FindByID(ctx context.Context, id uint64) (*model.Order, error) {
	var order model.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// FindByOrderNo loads one order and its items by business order number.
// โหลดคำสั่งซื้อพร้อม items จากเลข order ที่ใช้ในเชิงธุรกิจ
func (r *orderRepository) FindByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	var order model.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("order_no = ?", orderNo).
		First(&order).Error
	
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// FindByIdempotencyKey loads one order and its items by idempotency key.
// โหลดคำสั่งซื้อพร้อม items จาก idempotency key
func (r *orderRepository) FindByIdempotencyKey(ctx context.Context, key string) (*model.Order, error) {
	var order model.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("idempotency_key = ?", key).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// List returns paginated orders filtered by status, owner, date range, and sort order.
// คืนรายการคำสั่งซื้อแบบแบ่งหน้าตาม status, ผู้ใช้, ช่วงเวลา และลำดับ sort
func (r *orderRepository) List(ctx context.Context, status *string, userID *uint64, createdFrom, createdTo *time.Time, page, limit int, sort *string) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Order{})
	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if createdFrom != nil {
		query = query.Where("created_at >= ?", *createdFrom)
	}
	if createdTo != nil {
		query = query.Where("created_at <= ?", *createdTo)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if sort != nil && *sort != "" {
		query = query.Order(*sort)
	} else {
		query = query.Order("id desc")
	}

	offset := (page - 1) * limit
	if err := query.Preload("Items").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}
