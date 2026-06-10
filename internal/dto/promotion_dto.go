package dto

import (
	"encoding/json"
	"time"
)

type PromotionTargetRequest struct {
	TargetType  string  `json:"targetType" validate:"required"`
	TargetID    *uint64 `json:"targetId"`
	TargetValue *string `json:"targetValue"`
}

type PromotionConditionRequest struct {
	ConditionType   string          `json:"conditionType" validate:"required"`
	Operator        string          `json:"operator" validate:"required"`
	ValueJSON       json.RawMessage `json:"valueJson" validate:"required"`
	GroupKey        *string         `json:"groupKey"`
	LogicalOperator string          `json:"logicalOperator"`
}

type PromotionActionRequest struct {
	ActionType        string          `json:"actionType" validate:"required"`
	ValueAmount       *int64          `json:"valueAmount"`
	ValueBasisPoints  *int            `json:"valueBasisPoints"`
	ValueJSON         json.RawMessage `json:"valueJson"`
	MaxDiscountAmount *int64          `json:"maxDiscountAmount"`
	AppliesTo         string          `json:"appliesTo" validate:"required"`
}

type PromotionCreateRequest struct {
	Code            string                     `json:"code" validate:"required"`
	Name            string                     `json:"name" validate:"required"`
	Description     *string                    `json:"description"`
	Scope           string                     `json:"scope" validate:"required"`
	Priority        int                        `json:"priority"`
	Stackable       bool                       `json:"stackable"`
	Exclusive       bool                       `json:"exclusive"`
	StopProcessing  bool                       `json:"stopProcessing"`
	ConflictGroup   *string                    `json:"conflictGroup"`
	StartsAt        time.Time                  `json:"startsAt" validate:"required"`
	EndsAt          time.Time                  `json:"endsAt" validate:"required"`
	MaxUsage        *int                       `json:"maxUsage"`
	MaxUsagePerUser *int                       `json:"maxUsagePerUser"`
	Targets         []PromotionTargetRequest   `json:"targets"`
	Conditions      []PromotionConditionRequest `json:"conditions"`
	Actions         []PromotionActionRequest   `json:"actions"`
}

type PromotionReplaceRequest struct {
	Code            string                     `json:"code" validate:"required"`
	Name            string                     `json:"name" validate:"required"`
	Description     *string                    `json:"description"`
	Scope           string                     `json:"scope" validate:"required"`
	Priority        int                        `json:"priority"`
	Stackable       bool                       `json:"stackable"`
	Exclusive       bool                       `json:"exclusive"`
	StopProcessing  bool                       `json:"stopProcessing"`
	ConflictGroup   *string                    `json:"conflictGroup"`
	StartsAt        time.Time                  `json:"startsAt" validate:"required"`
	EndsAt          time.Time                  `json:"endsAt" validate:"required"`
	MaxUsage        *int                       `json:"maxUsage"`
	MaxUsagePerUser *int                       `json:"maxUsagePerUser"`
	Targets         []PromotionTargetRequest   `json:"targets"`
	Conditions      []PromotionConditionRequest `json:"conditions"`
	Actions         []PromotionActionRequest   `json:"actions"`
	ExpectedVersion  int                        `json:"expectedVersion"`
}

type PromotionPatchRequest struct {
	Name            *string    `json:"name"`
	Description     *string    `json:"description"`
	Priority        *int       `json:"priority"`
	StartsAt        *time.Time `json:"startsAt"`
	EndsAt          *time.Time `json:"endsAt"`
	ExpectedVersion  int        `json:"expectedVersion"`
}

type PromotionValidateRequest struct {
	ExpectedVersion int `json:"expectedVersion"`
}

type PromotionActivateRequest struct {
	ExpectedVersion int `json:"expectedVersion"`
}

type PromotionDeactivateRequest struct {
	ExpectedVersion int    `json:"expectedVersion"`
	Reason         string `json:"reason"`
}

type PromotionSummaryResponse struct {
	PromotionID    uint64     `json:"promotionId"`
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	Scope          string     `json:"scope"`
	Status         string     `json:"status"`
	Priority       int        `json:"priority"`
	StartsAt       time.Time  `json:"startsAt"`
	EndsAt         time.Time  `json:"endsAt"`
	Version        int        `json:"version"`
	Stackable      bool       `json:"stackable"`
	Exclusive      bool       `json:"exclusive"`
	StopProcessing bool       `json:"stopProcessing"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type PromotionDetailResponse struct {
	PromotionSummaryResponse
	Description     *string                    `json:"description,omitempty"`
	ConflictGroup   *string                    `json:"conflictGroup,omitempty"`
	MaxUsage        *int                       `json:"maxUsage,omitempty"`
	MaxUsagePerUser *int                       `json:"maxUsagePerUser,omitempty"`
	Targets         []PromotionTargetRequest   `json:"targets"`
	Conditions      []PromotionConditionRequest `json:"conditions"`
	Actions         []PromotionActionRequest   `json:"actions"`
}

type PromotionValidationResponse struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

type PromotionUsageListItem struct {
	PromotionID    uint64    `json:"promotionId"`
	UserID         *uint64   `json:"userId,omitempty"`
	OrderID        *uint64   `json:"orderId,omitempty"`
	UsageCount     int       `json:"usageCount"`
	DiscountAmount int64     `json:"discountAmount"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type PromotionUsageResponse struct {
	PromotionID         uint64                   `json:"promotionId"`
	TotalUsage          int64                    `json:"totalUsage"`
	TotalDiscountAmount int64                    `json:"totalDiscountAmount"`
	Items               []PromotionUsageListItem `json:"items"`
}

type PromotionListQuery struct {
	Status     *string    `query:"status"`
	Scope      *string    `query:"scope"`
	ActionType *string    `query:"actionType"`
	Code       *string    `query:"code"`
	ActiveAt   *time.Time `query:"activeAt"`
	Page       int        `query:"page"`
	Limit      int        `query:"limit"`
	Sort       *string    `query:"sort"`
}

type PromotionListResponse struct {
	Items      []PromotionSummaryResponse `json:"items"`
	Pagination Pagination                 `json:"pagination"`
}

type PromotionUsageQuery struct {
	UserID *uint64    `query:"userId"`
	From   *time.Time `query:"from"`
	To     *time.Time `query:"to"`
	Page   int        `query:"page"`
	Limit  int        `query:"limit"`
}
