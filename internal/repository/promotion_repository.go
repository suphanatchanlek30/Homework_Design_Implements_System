package repository

import (
	"context"
	"time"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"gorm.io/gorm"
)

type PromotionRepository interface {
	FindActivePromotions(ctx context.Context, now time.Time) ([]model.Promotion, error)
	FindByID(ctx context.Context, id uint64) (*model.Promotion, error)
	// Add other specific methods as needed
}

type promotionRepository struct {
	*BaseRepository[model.Promotion]
}

func NewPromotionRepository(db *gorm.DB) PromotionRepository {
	return &promotionRepository{
		BaseRepository: NewBaseRepository[model.Promotion](db),
	}
}

func (r *promotionRepository) FindActivePromotions(ctx context.Context, now time.Time) ([]model.Promotion, error) {
	var promotions []model.Promotion
	
	// Querying active promotions:
	// 1. Status is ACTIVE
	// 2. now is between starts_at and ends_at
	// 3. Preload Conditions and Actions as they are needed for calculation
	// 4. Sort by scope priority (if we had a scope order table) and then promotion priority
	
	err := r.db.WithContext(ctx).
		Preload("Conditions").
		Preload("Actions").
		Preload("Targets").
		Where("status = ?", "ACTIVE").
		Where("starts_at <= ? AND ends_at >= ?", now, now).
		Order("priority ASC, created_at ASC").
		Find(&promotions).Error

	if err != nil {
		return nil, err
	}

	return promotions, nil
}

func (r *promotionRepository) FindByID(ctx context.Context, id uint64) (*model.Promotion, error) {
	var promotion model.Promotion
	err := r.db.WithContext(ctx).
		Preload("Conditions").
		Preload("Actions").
		Preload("Targets").
		First(&promotion, id).Error
	
	if err != nil {
		return nil, err
	}
	return &promotion, nil
}
