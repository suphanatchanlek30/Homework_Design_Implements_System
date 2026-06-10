package repository

import (
	"context"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	FindByID(ctx context.Context, id uint64) (*model.Order, error)
	FindByOrderNo(ctx context.Context, orderNo string) (*model.Order, error)
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
