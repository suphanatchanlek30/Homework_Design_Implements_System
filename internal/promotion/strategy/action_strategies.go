package strategy

import (
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
)

type PercentageDiscountAction struct{}

// Execute calculates the discount produced by a percentage-based legacy action strategy.
// คำนวณส่วนลดจาก strategy แบบเปอร์เซ็นต์ในชุดโค้ด legacy
func (a *PercentageDiscountAction) Execute(order *model.Order, action model.PromotionAction) int64 {
	if action.ActionType != "PERCENTAGE_DISCOUNT" && action.ActionType != "CART_PERCENTAGE_DISCOUNT" {
		return 0
	}

	if action.ValueBasisPoints == nil {
		return 0
	}

	basisPoints := int64(*action.ValueBasisPoints)

	var discount int64
	if action.AppliesTo == "CART" || action.ActionType == "CART_PERCENTAGE_DISCOUNT" {
		// Calculate based on order total
		discount = (order.OriginalTotal * basisPoints) / 10000
	} else {
		// This should technically apply to specific items,
		// but for a simple engine, let's assume it applies to the current base total
		// or we filter targets elsewhere.
		// For now, let's calculate based on the total for demonstration.
		discount = (order.OriginalTotal * basisPoints) / 10000
	}

	// Handle Max Discount Amount
	if action.MaxDiscountAmount != nil && discount > *action.MaxDiscountAmount {
		discount = *action.MaxDiscountAmount
	}

	return discount
}

type FixedAmountAction struct{}

// Execute calculates the discount produced by a fixed-amount legacy action strategy.
// คำนวณส่วนลดจาก strategy แบบจำนวนเงินคงที่ในชุดโค้ด legacy
func (a *FixedAmountAction) Execute(order *model.Order, action model.PromotionAction) int64 {
	if action.ActionType != "FIXED_AMOUNT_DISCOUNT" && action.ActionType != "CART_FIXED_AMOUNT_DISCOUNT" {
		return 0
	}

	if action.ValueAmount == nil {
		return 0
	}

	discount := *action.ValueAmount

	// Ensure discount doesn't exceed order total
	if discount > order.OriginalTotal {
		discount = order.OriginalTotal
	}

	return discount
}
