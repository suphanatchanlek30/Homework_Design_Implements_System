package dto

import "time"

type CalculationLogQuery struct {
	RequestID   *string    `query:"requestId"`
	OrderID     *uint64    `query:"orderId"`
	UserID      *uint64    `query:"userId"`
	PromotionID *uint64    `query:"promotionId"`
	CreatedFrom *time.Time `query:"createdFrom"`
	CreatedTo   *time.Time `query:"createdTo"`
	Page        int        `query:"page"`
	Limit       int        `query:"limit"`
	Sort        *string    `query:"sort"`
}

type CalculationLogSummaryResponse struct {
	CalculationID          string    `json:"calculationId"`
	OrderID                *uint64   `json:"orderId,omitempty"`
	RequestID              string    `json:"requestId"`
	UserID                 *uint64   `json:"userId,omitempty"`
	OriginalTotal          int64     `json:"originalTotal"`
	DiscountTotal          int64     `json:"discountTotal"`
	FinalTotal             int64     `json:"finalTotal"`
	AppliedPromotionCount   int       `json:"appliedPromotionCount"`
	SkippedPromotionCount   int       `json:"skippedPromotionCount"`
	CreatedAt              time.Time `json:"createdAt"`
}

type CalculationLogListResponse struct {
	Items      []CalculationLogSummaryResponse `json:"items"`
	Pagination Pagination                      `json:"pagination"`
}

type CalculationLogDetailResponse struct {
	CalculationLogSummaryResponse
	AppliedPromotions  []PricingPromotionAppliedResponse `json:"appliedPromotions"`
	SkippedPromotions  []PricingPromotionSkippedResponse `json:"skippedPromotions"`
	CalculationSnapshot map[string]any                  `json:"calculationSnapshot,omitempty"`
}

type CalculationLogReplayRequest struct {
	Mode string `json:"mode"`
}

type CalculationLogReplayResponse struct {
	CalculationID string                   `json:"calculationId"`
	Mode          string                   `json:"mode"`
	OriginalResult PricingResultResponse    `json:"originalResult"`
	ReplayResult  PricingResultResponse     `json:"replayResult"`
	Matched       bool                     `json:"matched"`
	Differences   []string                 `json:"differences"`
}
