package repository

import (
	"context"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"gorm.io/gorm"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *model.ProductCategory) error
	Update(ctx context.Context, category *model.ProductCategory) error
	FindByID(ctx context.Context, id uint64) (*model.ProductCategory, error)
	FindByNameAndParent(ctx context.Context, name string, parentID *uint64) (*model.ProductCategory, error)
	List(ctx context.Context, status *string, parentID *uint64, keyword *string, page, limit int, sort *string) ([]model.ProductCategory, int64, error)
	FindDescendants(ctx context.Context, categoryID uint64) ([]model.ProductCategory, error)
}

type categoryRepository struct {
	*BaseRepository[model.ProductCategory]
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{
		BaseRepository: NewBaseRepository[model.ProductCategory](db),
		db:             db,
	}
}

func (r *categoryRepository) List(ctx context.Context, status *string, parentID *uint64, keyword *string, page, limit int, sort *string) ([]model.ProductCategory, int64, error) {
	var categories []model.ProductCategory
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ProductCategory{})

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}
	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
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
	if err := query.Offset(offset).Limit(limit).Find(&categories).Error; err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

func (r *categoryRepository) FindByNameAndParent(ctx context.Context, name string, parentID *uint64) (*model.ProductCategory, error) {
	var category model.ProductCategory

	query := r.db.WithContext(ctx).Where("name = ?", name)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	if err := query.First(&category).Error; err != nil {
		return nil, err
	}

	return &category, nil
}

// FindDescendants recursively finds all descendants to prevent circular hierarchy
func (r *categoryRepository) FindDescendants(ctx context.Context, categoryID uint64) ([]model.ProductCategory, error) {
	// A simpler approach for MySQL 8+ is recursive CTE, but here we can just use preload or recursive function.
	// For simplicity in this example without recursive CTE, we might just query all and build tree, 
	// or recursively query. Let's do a basic recursive query or simple check.
	// Actually, CTE is supported in MySQL 8.0:
	var descendants []model.ProductCategory
	cteQuery := `
	WITH RECURSIVE CategoryPath AS (
		SELECT id, parent_id, name FROM product_categories WHERE parent_id = ?
		UNION ALL
		SELECT c.id, c.parent_id, c.name FROM product_categories c
		INNER JOIN CategoryPath cp ON c.parent_id = cp.id
	)
	SELECT * FROM CategoryPath;
	`
	if err := r.db.WithContext(ctx).Raw(cteQuery, categoryID).Scan(&descendants).Error; err != nil {
		return nil, err
	}
	return descendants, nil
}
