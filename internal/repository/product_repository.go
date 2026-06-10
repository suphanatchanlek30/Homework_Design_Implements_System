package repository

import (
	"context"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"gorm.io/gorm"
)

type ProductRepository interface {
	FindByID(ctx context.Context, id uint64) (*model.Product, error)
	FindBySKU(ctx context.Context, sku string) (*model.Product, error)
	ListActive(ctx context.Context) ([]model.Product, error)
}

type productRepository struct {
	*BaseRepository[model.Product]
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{
		BaseRepository: NewBaseRepository[model.Product](db),
	}
}

func (r *productRepository) FindBySKU(ctx context.Context, sku string) (*model.Product, error) {
	var product model.Product
	if err := r.db.WithContext(ctx).Where("sku = ?", sku).First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) ListActive(ctx context.Context) ([]model.Product, error) {
	var products []model.Product
	if err := r.db.WithContext(ctx).Where("status = ?", "ACTIVE").Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}
