package repository

import (
	"context"
	"time"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"gorm.io/gorm"
)

type PromotionSummary struct {
	ID            uint64    `json:"id"`
	Code          *string   `json:"code"`
	Name          string    `json:"name"`
	Scope         string    `json:"scope"`
	Status        string    `json:"status"`
	Priority      int       `json:"priority"`
	StartsAt      time.Time `json:"startsAt"`
	EndsAt        time.Time `json:"endsAt"`
	Version       int       `json:"version"`
	Stackable     bool      `json:"stackable"`
	Exclusive     bool      `json:"exclusive"`
	StopProcessing bool      `json:"stopProcessing"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type PromotionListFilter struct {
	Status     *string
	Scope      *string
	ActionType *string
	Code       *string
	ActiveAt   *time.Time
}

type PromotionRepository interface {
	Create(ctx context.Context, promotion *model.Promotion) error
	Update(ctx context.Context, promotion *model.Promotion) error
	FindActivePromotions(ctx context.Context, now time.Time) ([]model.Promotion, error)
	FindByID(ctx context.Context, id uint64) (*model.Promotion, error)
	FindByCode(ctx context.Context, code string) (*model.Promotion, error)
	List(ctx context.Context, filter PromotionListFilter, page, limit int, sort *string) ([]PromotionSummary, int64, error)
	FindUsages(ctx context.Context, promotionID uint64, userID *uint64, from, to *time.Time, page, limit int) ([]model.PromotionUsage, int64, error)
}

type promotionRepository struct {
	*BaseRepository[model.Promotion]
	db *gorm.DB
}

func NewPromotionRepository(db *gorm.DB) PromotionRepository {
	return &promotionRepository{
		BaseRepository: NewBaseRepository[model.Promotion](db),
		db:             db,
	}
}

func (r *promotionRepository) Create(ctx context.Context, promotion *model.Promotion) error {
	return r.db.WithContext(ctx).Create(promotion).Error
}

func (r *promotionRepository) Update(ctx context.Context, promotion *model.Promotion) error {
	return r.db.WithContext(ctx).Save(promotion).Error
}

func (r *promotionRepository) FindActivePromotions(ctx context.Context, now time.Time) ([]model.Promotion, error) {
	var promotions []model.Promotion

	err := r.db.WithContext(ctx).
		Preload("Conditions").
		Preload("Actions").
		Preload("Targets").
		Where("status = ?", "ACTIVE").
		Where("starts_at <= ? AND ends_at >= ?", now, now).
		Order("scope ASC, priority ASC, created_at ASC, id ASC").
		Find(&promotions).Error
	if err != nil {
		return nil, err
	}

	return promotions, nil
}

func (r *promotionRepository) FindByID(ctx context.Context, id uint64) (*model.Promotion, error) {
	var promotion model.Promotion
	if err := r.db.WithContext(ctx).
		Preload("Conditions").
		Preload("Actions").
		Preload("Targets").
		First(&promotion, id).Error; err != nil {
		return nil, err
	}
	return &promotion, nil
}

func (r *promotionRepository) FindByCode(ctx context.Context, code string) (*model.Promotion, error) {
	var promotion model.Promotion
	if err := r.db.WithContext(ctx).
		Preload("Conditions").
		Preload("Actions").
		Preload("Targets").
		Where("code = ?", code).
		First(&promotion).Error; err != nil {
		return nil, err
	}
	return &promotion, nil
}

func (r *promotionRepository) List(ctx context.Context, filter PromotionListFilter, page, limit int, sort *string) ([]PromotionSummary, int64, error) {
	var summaries []PromotionSummary
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Promotion{})

	if filter.Status != nil && *filter.Status != "" {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Scope != nil && *filter.Scope != "" {
		query = query.Where("scope = ?", *filter.Scope)
	}
	if filter.Code != nil && *filter.Code != "" {
		query = query.Where("code = ?", *filter.Code)
	}
	if filter.ActiveAt != nil {
		query = query.Where("starts_at <= ? AND ends_at >= ?", *filter.ActiveAt, *filter.ActiveAt)
	}
	if filter.ActionType != nil && *filter.ActionType != "" {
		query = query.Where(
			"EXISTS (SELECT 1 FROM promotion_actions pa WHERE pa.promotion_id = promotions.id AND pa.action_type = ?)",
			*filter.ActionType,
		)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if sort != nil && *sort != "" {
		query = query.Order(*sort)
	} else {
		query = query.Order("scope ASC, priority ASC, created_at ASC, id ASC")
	}

	offset := (page - 1) * limit
	if err := query.
		Select("promotions.id, promotions.code, promotions.name, promotions.scope, promotions.status, promotions.priority, promotions.starts_at, promotions.ends_at, promotions.version, promotions.stackable, promotions.exclusive, promotions.stop_processing, promotions.created_at, promotions.updated_at").
		Offset(offset).
		Limit(limit).
		Scan(&summaries).Error; err != nil {
		return nil, 0, err
	}

	return summaries, total, nil
}

func (r *promotionRepository) FindUsages(ctx context.Context, promotionID uint64, userID *uint64, from, to *time.Time, page, limit int) ([]model.PromotionUsage, int64, error) {
	var usages []model.PromotionUsage
	var total int64

	query := r.db.WithContext(ctx).Model(&model.PromotionUsage{}).Where("promotion_id = ?", promotionID)
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if from != nil {
		query = query.Where("created_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("created_at <= ?", *to)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&usages).Error; err != nil {
		return nil, 0, err
	}

	return usages, total, nil
}
