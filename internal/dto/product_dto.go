package dto

import "time"

type CreateProductRequest struct {
	SKU         string `json:"sku" validate:"required"`
	Name        string `json:"name" validate:"required"`
	CategoryID  uint64 `json:"categoryId" validate:"required"`
	PriceAmount int64  `json:"priceAmount" validate:"required,min=0"`
	Currency    string `json:"currency" validate:"required"`
	Status      string `json:"status" validate:"required,oneof=ACTIVE INACTIVE"`
}

type UpdateProductRequest struct {
	Name        *string `json:"name"`
	CategoryID  *uint64 `json:"categoryId"`
	PriceAmount *int64  `json:"priceAmount" validate:"omitempty,min=0"`
	Currency    *string `json:"currency"`
	Status      *string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

type ProductResponse struct {
	ID          uint64    `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	CategoryID  uint64    `json:"categoryId"`
	PriceAmount int64     `json:"priceAmount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ProductQuery struct {
	Status     *string `query:"status"`
	CategoryID *string `query:"categoryId"`
	SKU        *string `query:"sku"`
	Keyword    *string `query:"keyword"`
	Page       int     `query:"page"`
	Limit      int     `query:"limit"`
	Sort       *string `query:"sort"`
}

type ProductListResponse struct {
	Items      []ProductResponse `json:"items"`
	Pagination Pagination        `json:"pagination"`
}
