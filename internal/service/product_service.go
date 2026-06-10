package service

import (
	"context"
	"errors"
	"math"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrProductNotFound      = errors.New("product not found")
	ErrSKUAlreadyExists     = errors.New("sku already exists")
	ErrInvalidPriceAmount   = errors.New("invalid price amount: must be >= 0")
	ErrUnsupportedCurrency  = errors.New("unsupported currency")
)

type ProductService interface {
	Create(ctx context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error)
	Update(ctx context.Context, id uint64, req dto.UpdateProductRequest) (*dto.ProductResponse, error)
	GetByID(ctx context.Context, id uint64) (*dto.ProductResponse, error)
	List(ctx context.Context, query dto.ProductQuery) (*dto.ProductListResponse, error)
}

type productService struct {
	repo         repository.ProductRepository
	categoryRepo repository.CategoryRepository
}

func NewProductService(repo repository.ProductRepository, categoryRepo repository.CategoryRepository) ProductService {
	return &productService{
		repo:         repo,
		categoryRepo: categoryRepo,
	}
}

func (s *productService) Create(ctx context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error) {
	if req.SKU == "" {
		return nil, errors.New("sku is required")
	}
	if req.PriceAmount < 0 {
		return nil, ErrInvalidPriceAmount
	}
	if req.Currency != "THB" {
		return nil, ErrUnsupportedCurrency
	}

	// Check SKU uniqueness
	existing, err := s.repo.FindBySKU(ctx, req.SKU)
	if err == nil && existing != nil {
		return nil, ErrSKUAlreadyExists
	}

	// Check Category existence
	_, err = s.categoryRepo.FindByID(ctx, req.CategoryID)
	if err != nil {
		return nil, ErrParentNotFound // Using ErrParentNotFound as ErrCategoryNotFound
	}

	product := &model.Product{
		SKU:         req.SKU,
		Name:        req.Name,
		CategoryID:  req.CategoryID,
		PriceAmount: req.PriceAmount,
		Currency:    req.Currency,
		Status:      req.Status,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}

	return s.toResponse(product), nil
}

func (s *productService) Update(ctx context.Context, id uint64, req dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.PriceAmount != nil {
		if *req.PriceAmount < 0 {
			return nil, ErrInvalidPriceAmount
		}
		product.PriceAmount = *req.PriceAmount
	}
	if req.Currency != nil {
		if *req.Currency != "THB" {
			return nil, ErrUnsupportedCurrency
		}
		product.Currency = *req.Currency
	}
	if req.Status != nil {
		product.Status = *req.Status
	}
	if req.CategoryID != nil {
		_, err = s.categoryRepo.FindByID(ctx, *req.CategoryID)
		if err != nil {
			return nil, ErrParentNotFound
		}
		product.CategoryID = *req.CategoryID
	}

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}

	return s.toResponse(product), nil
}

func (s *productService) GetByID(ctx context.Context, id uint64) (*dto.ProductResponse, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrProductNotFound
	}
	return s.toResponse(product), nil
}

func (s *productService) List(ctx context.Context, query dto.ProductQuery) (*dto.ProductListResponse, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 || query.Limit > 100 {
		query.Limit = 10
	}

	products, total, err := s.repo.List(ctx, query.Status, query.CategoryID, query.SKU, query.Keyword, query.Page, query.Limit, query.Sort)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ProductResponse, len(products))
	for i, p := range products {
		items[i] = *s.toResponse(&p)
	}

	totalPages := int(math.Ceil(float64(total) / float64(query.Limit)))

	return &dto.ProductListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:       query.Page,
			Limit:      query.Limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *productService) toResponse(p *model.Product) *dto.ProductResponse {
	return &dto.ProductResponse{
		ID:          p.ID,
		SKU:         p.SKU,
		Name:        p.Name,
		CategoryID:  p.CategoryID,
		PriceAmount: p.PriceAmount,
		Currency:    p.Currency,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
