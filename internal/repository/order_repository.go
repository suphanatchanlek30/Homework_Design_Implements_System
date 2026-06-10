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

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{
		BaseRepository: NewBaseRepository[model.Order](db),
	}
}

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
