package dto

import "time"

type CreateCategoryRequest struct {
	Name     string  `json:"name" validate:"required"`
	ParentID *uint64 `json:"parentId"`
	Status   string  `json:"status" validate:"required,oneof=ACTIVE INACTIVE"`
}

type UpdateCategoryRequest struct {
	Name     *string `json:"name"`
	ParentID *uint64 `json:"parentId"`
	Status   *string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

type CategoryResponse struct {
	ID        uint64     `json:"id"`
	Name      string     `json:"name"`
	ParentID  *uint64    `json:"parentId,omitempty"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type CategoryQuery struct {
	Status   *string `query:"status"`
	ParentID *uint64 `query:"parentId"`
	Keyword  *string `query:"keyword"`
	Page     int     `query:"page"`
	Limit    int     `query:"limit"`
	Sort     *string `query:"sort"`
}

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

type CategoryListResponse struct {
	Items      []CategoryResponse `json:"items"`
	Pagination Pagination         `json:"pagination"`
}
