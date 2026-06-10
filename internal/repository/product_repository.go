package repository

import (
	"context"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"gorm.io/gorm"
)

type ProductRepository interface {
	Create(ctx context.Context, product *model.Product) error
	Update(ctx context.Context, product *model.Product) error
	FindByID(ctx context.Context, id uint64) (*model.Product, error)
	FindBySKU(ctx context.Context, sku string) (*model.Product, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]model.Product, error)
	List(ctx context.Context, status, categoryID, sku, keyword *string, page, limit int, sort *string) ([]model.Product, int64, error)
}

type productRepository struct {
	*BaseRepository[model.Product]
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{
		BaseRepository: NewBaseRepository[model.Product](db),
		db:             db,
	}
}

func (r *productRepository) FindBySKU(ctx context.Context, sku string) (*model.Product, error) {
	var product model.Product
	if err := r.db.WithContext(ctx).Where("sku = ?", sku).First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) FindByIDs(ctx context.Context, ids []uint64) ([]model.Product, error) {
	var products []model.Product
	if len(ids) == 0 {
		return products, nil
	}

	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (r *productRepository) List(ctx context.Context, status, categoryID, sku, keyword *string, page, limit int, sort *string) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Product{})

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}
	if categoryID != nil && *categoryID != "" {
		query = query.Where("category_id = ?", *categoryID)
	}
	if sku != nil && *sku != "" {
		query = query.Where("sku = ?", *sku)
	}
	if keyword != nil && *keyword != "" {
		query = query.Where("name LIKE ?", "%"+*keyword+"%")
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
	if err := query.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}
