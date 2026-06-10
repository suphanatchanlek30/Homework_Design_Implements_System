package service

import (
	"context"
	"errors"
	"math"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
)

var (
	ErrCategoryNotFound       = errors.New("category not found")
	ErrParentNotFound         = errors.New("parent category not found")
	ErrCircularHierarchy      = errors.New("invalid category hierarchy: circular reference detected")
	ErrCategoryAlreadyExists  = errors.New("category already exists")
	ErrCategoryUpdateConflict = errors.New("category update conflict")
	ErrInvalidCategoryName    = errors.New("category name is required")
	ErrInvalidCategoryStatus  = errors.New("invalid status: must be ACTIVE or INACTIVE")
)

type CategoryService interface {
	Create(ctx context.Context, req dto.CreateCategoryRequest) (*dto.CategoryResponse, error)
	Update(ctx context.Context, id uint64, req dto.UpdateCategoryRequest) (*dto.CategoryResponse, error)
	GetByID(ctx context.Context, id uint64) (*dto.CategoryResponse, error)
	List(ctx context.Context, query dto.CategoryQuery) (*dto.CategoryListResponse, error)
}

type categoryService struct {
	repo repository.CategoryRepository
}

func NewCategoryService(repo repository.CategoryRepository) CategoryService {
	return &categoryService{repo: repo}
}

func (s *categoryService) Create(ctx context.Context, req dto.CreateCategoryRequest) (*dto.CategoryResponse, error) {
	if req.Name == "" {
		return nil, ErrInvalidCategoryName
	}
	if req.Status != "ACTIVE" && req.Status != "INACTIVE" {
		return nil, ErrInvalidCategoryStatus
	}

	if req.ParentID != nil {
		_, err := s.repo.FindByID(ctx, *req.ParentID)
		if err != nil {
			return nil, ErrParentNotFound
		}
	}

	existing, err := s.repo.FindByNameAndParent(ctx, req.Name, req.ParentID)
	if err == nil && existing != nil {
		return nil, ErrCategoryAlreadyExists
	}

	category := &model.ProductCategory{
		Name:     req.Name,
		ParentID: req.ParentID,
		Status:   req.Status,
	}

	if err := s.repo.Create(ctx, category); err != nil {
		return nil, err
	}

	return s.toResponse(category), nil
}

func (s *categoryService) Update(ctx context.Context, id uint64, req dto.UpdateCategoryRequest) (*dto.CategoryResponse, error) {
	category, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	if req.Name != nil {
		if *req.Name == "" {
			return nil, ErrInvalidCategoryName
		}
		category.Name = *req.Name
	}
	if req.Status != nil {
		if *req.Status != "ACTIVE" && *req.Status != "INACTIVE" {
			return nil, ErrInvalidCategoryStatus
		}
		category.Status = *req.Status
	}

	if req.ParentID != nil {
		if *req.ParentID == id {
			return nil, ErrCircularHierarchy
		}

		// Check if parent exists
		_, err := s.repo.FindByID(ctx, *req.ParentID)
		if err != nil {
			return nil, ErrParentNotFound
		}

		// Check for circular hierarchy
		descendants, err := s.repo.FindDescendants(ctx, id)
		if err != nil {
			return nil, err
		}

		for _, d := range descendants {
			if d.ID == *req.ParentID {
				return nil, ErrCircularHierarchy
			}
		}

		category.ParentID = req.ParentID
	}

	existing, err := s.repo.FindByNameAndParent(ctx, category.Name, category.ParentID)
	if err == nil && existing != nil && existing.ID != id {
		return nil, ErrCategoryUpdateConflict
	}

	if err := s.repo.Update(ctx, category); err != nil {
		return nil, err
	}

	return s.toResponse(category), nil
}

func (s *categoryService) GetByID(ctx context.Context, id uint64) (*dto.CategoryResponse, error) {
	category, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}
	return s.toResponse(category), nil
}

func (s *categoryService) List(ctx context.Context, query dto.CategoryQuery) (*dto.CategoryListResponse, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 || query.Limit > 100 {
		query.Limit = 10
	}

	categories, total, err := s.repo.List(ctx, query.Status, query.ParentID, query.Keyword, query.Page, query.Limit, query.Sort)
	if err != nil {
		return nil, err
	}

	items := make([]dto.CategoryResponse, len(categories))
	for i, c := range categories {
		items[i] = *s.toResponse(&c)
	}

	totalPages := int(math.Ceil(float64(total) / float64(query.Limit)))

	return &dto.CategoryListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:       query.Page,
			Limit:      query.Limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *categoryService) toResponse(c *model.ProductCategory) *dto.CategoryResponse {
	return &dto.CategoryResponse{
		ID:        c.ID,
		Name:      c.Name,
		ParentID:  c.ParentID,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
