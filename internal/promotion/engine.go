package promotion

import (
	"context"
	"time"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
)

type CalculationResult struct {
	OriginalTotal     int64
	DiscountTotal     int64
	FinalTotal        int64
	AppliedPromotions []AppliedPromotion
}

type AppliedPromotion struct {
	PromotionID    uint64
	PromotionName  string
	DiscountAmount int64
}

type PromotionEngine interface {
	Calculate(ctx context.Context, order *model.Order) (*CalculationResult, error)
}

type promotionEngine struct {
	promoRepo repository.PromotionRepository
}

func NewPromotionEngine(promoRepo repository.PromotionRepository) PromotionEngine {
	return &promotionEngine{
		promoRepo: promoRepo,
	}
}

func (e *promotionEngine) Calculate(ctx context.Context, order *model.Order) (*CalculationResult, error) {
	now := time.Now()
	promos, err := e.promoRepo.FindActivePromotions(ctx, now)
	if err != nil {
		return nil, err
	}

	// 1. Group and Sort Promotions by Scope
	// ITEM -> CART -> COUPON -> SHIPPING
	// Within each scope, they are already sorted by priority from the repository

	result := &CalculationResult{
		OriginalTotal:     order.OriginalTotal,
		AppliedPromotions: []AppliedPromotion{},
	}

	// Current totals that will be updated through the pipeline
	currentDiscount := int64(0)

	// Implementation note:
	// To handle ITEM level discounts properly, we might need to track individual item prices.
	// For now, let's process them sequentially.

	for _, promo := range promos {
		// TODO: Implement Condition Evaluator
		// if !e.evaluateConditions(promo.Conditions, order) { continue }

		// TODO: Implement Action Strategy execution
		// discount := e.executeActions(promo.Actions, order)

		// Placeholder logic for now
		discount := int64(0)

		if discount > 0 {
			currentDiscount += discount
			result.AppliedPromotions = append(result.AppliedPromotions, AppliedPromotion{
				PromotionID:    promo.ID,
				PromotionName:  promo.Name,
				DiscountAmount: discount,
			})

			if promo.StopProcessing {
				break
			}
		}
	}

	result.DiscountTotal = currentDiscount
	result.FinalTotal = result.OriginalTotal - result.DiscountTotal
	if result.FinalTotal < 0 {
		result.FinalTotal = 0
	}

	return result, nil
}
