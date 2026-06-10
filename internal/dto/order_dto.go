package dto

import "time"

type OrderItemRequest struct {
	ProductID uint64 `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type OrderShippingRequest struct {
	Method string `json:"method"`
}

type OrderConfirmRequest struct {
	CalculationID     string               `json:"calculationId"`
	AcceptedFinalTotal int64                `json:"acceptedFinalTotal"`
	UserID            *uint64              `json:"userId"`
	Items             []OrderItemRequest   `json:"items"`
	CouponCodes       []string             `json:"couponCodes"`
	PaymentMethod     *string              `json:"paymentMethod"`
	Shipping          *OrderShippingRequest `json:"shipping"`
	Currency          string               `json:"currency"`
}

type OrderItemResponse struct {
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

type OrderSummaryResponse struct {
	OrderID          uint64                        `json:"orderId"`
	OrderNo          string                        `json:"orderNo"`
	UserID           *uint64                       `json:"userId,omitempty"`
	Status           string                        `json:"status"`
	Currency         string                        `json:"currency"`
	OriginalTotal    int64                         `json:"originalTotal"`
	DiscountTotal    int64                         `json:"discountTotal"`
	FinalTotal       int64                         `json:"finalTotal"`
	CalculationID    string                        `json:"calculationId"`
	CreatedAt        time.Time                     `json:"createdAt"`
	UpdatedAt        time.Time                     `json:"updatedAt"`
}

type OrderDetailResponse struct {
	OrderSummaryResponse
	Items             []OrderItemResponse            `json:"items"`
	AppliedPromotions []PricingPromotionAppliedResponse `json:"appliedPromotions"`
	SkippedPromotions []PricingPromotionSkippedResponse `json:"skippedPromotions"`
	CalculationSnapshot map[string]any               `json:"calculationSnapshot,omitempty"`
}

type OrderConfirmResponse struct {
	OrderDetailResponse
}

type OrderListQuery struct {
	Status     *string    `query:"status"`
	UserID     *uint64    `query:"userId"`
	CreatedFrom *time.Time `query:"createdFrom"`
	CreatedTo   *time.Time `query:"createdTo"`
	Page       int        `query:"page"`
	Limit      int        `query:"limit"`
	Sort       *string    `query:"sort"`
}

type OrderListResponse struct {
	Items      []OrderSummaryResponse `json:"items"`
	Pagination Pagination             `json:"pagination"`
}
