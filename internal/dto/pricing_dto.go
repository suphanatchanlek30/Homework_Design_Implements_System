package dto

import "time"

type PricingItemRequest struct {
	ProductID uint64 `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type PricingShippingRequest struct {
	Method string `json:"method"`
}

type PricingCalculateRequest struct {
	UserID        *uint64               `json:"userId"`
	Items         []PricingItemRequest  `json:"items"`
	CouponCodes   []string              `json:"couponCodes"`
	PaymentMethod *string               `json:"paymentMethod"`
	Shipping      *PricingShippingRequest `json:"shipping"`
	Currency      string                `json:"currency"`
}

type PricingItemResponse struct {
	ProductID      uint64    `json:"productId"`
	SKU            string    `json:"sku"`
	ProductName    string    `json:"productName"`
	Quantity       int       `json:"quantity"`
	UnitPrice      int64     `json:"unitPrice"`
	OriginalAmount int64     `json:"originalAmount"`
	DiscountAmount int64     `json:"discountAmount"`
	FinalAmount    int64     `json:"finalAmount"`
	CreatedAt      time.Time `json:"createdAt,omitempty"`
}

type PricingPromotionAppliedResponse struct {
	PromotionID    uint64 `json:"promotionId"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Scope          string `json:"scope"`
	DiscountAmount int64  `json:"discountAmount"`
}

type PricingPromotionSkippedResponse struct {
	PromotionID uint64 `json:"promotionId"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Reason      string `json:"reason"`
}

type PricingResultResponse struct {
	CalculationID    string                          `json:"calculationId"`
	OriginalTotal     int64                           `json:"originalTotal"`
	DiscountTotal     int64                           `json:"discountTotal"`
	FinalTotal        int64                           `json:"finalTotal"`
	Currency          string                          `json:"currency"`
	Items             []PricingItemResponse           `json:"items"`
	AppliedPromotions []PricingPromotionAppliedResponse `json:"appliedPromotions"`
	SkippedPromotions []PricingPromotionSkippedResponse `json:"skippedPromotions"`
}

