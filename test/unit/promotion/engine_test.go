package promotion_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/promotion"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
)

func TestCalculator_ItemAndCartPromotions(t *testing.T) {
	code1 := "P1_10"
	code2 := "P2_MINUS_100"
	code3 := "CART_5"
	conflictGroup := "group-1"

	calculator := promotion.NewCalculator()
	result, err := calculator.Calculate(nil, promotion.CalculationContext{
		Now: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Items: []promotion.CalculationItem{
			{ProductID: 1, SKU: "P1", ProductName: "Product 1", CategoryID: 1, Quantity: 1, UnitPrice: 100000},
			{ProductID: 2, SKU: "P2", ProductName: "Product 2", CategoryID: 1, Quantity: 1, UnitPrice: 50000},
		},
		Promotions: []model.Promotion{
			{
				BaseModel: model.BaseModel{ID: 1, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				Code:      &code1,
					Name:      "Product 1 10%",
					Scope:     "ITEM",
					Status:    "ACTIVE",
					Stackable: true,
					StartsAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					EndsAt:    time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
					Targets: []model.PromotionTarget{{TargetType: "PRODUCT", TargetID: uint64Ptr(1)}},
					Actions: []model.PromotionAction{{ActionType: "PERCENTAGE_DISCOUNT", ValueBasisPoints: intPtr(1000), AppliesTo: "ITEM"}},
			},
			{
				BaseModel: model.BaseModel{ID: 2, CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
				Code:      &code2,
					Name:      "Product 2 minus 100",
					Scope:     "ITEM",
					Status:    "ACTIVE",
					Stackable: true,
					StartsAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					EndsAt:    time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
					Targets: []model.PromotionTarget{{TargetType: "PRODUCT", TargetID: uint64Ptr(2)}},
					Actions: []model.PromotionAction{{ActionType: "FIXED_AMOUNT_DISCOUNT", ValueAmount: int64Ptr(100), AppliesTo: "ITEM"}},
			},
			{
				BaseModel: model.BaseModel{ID: 3, CreatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
				Code:      &code3,
					Name:      "Cart 5%",
					Scope:     "CART",
					Status:    "ACTIVE",
					Stackable: true,
					StartsAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					EndsAt:    time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
					ConflictGroup: &conflictGroup,
					Targets: []model.PromotionTarget{{TargetType: "CART"}},
				Actions: []model.PromotionAction{{ActionType: "CART_PERCENTAGE_DISCOUNT", ValueBasisPoints: intPtr(500), AppliesTo: "CART"}},
			},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(150000), result.OriginalTotal)
	assert.Greater(t, result.DiscountTotal, int64(0))
	assert.Equal(t, result.FinalTotal, result.OriginalTotal-result.DiscountTotal)
	assert.Len(t, result.AppliedPromotions, 3)
}

func TestCalculator_SkipsInactivePromotion(t *testing.T) {
	code := "OFF"
	calculator := promotion.NewCalculator()
	result, err := calculator.Calculate(nil, promotion.CalculationContext{
		Now: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Items: []promotion.CalculationItem{{ProductID: 1, SKU: "P1", ProductName: "P1", CategoryID: 1, Quantity: 1, UnitPrice: 100000}},
		Promotions: []model.Promotion{
			{
				BaseModel: model.BaseModel{ID: 1, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				Code:      &code,
					Name:      "Inactive",
					Scope:     "ITEM",
					Status:    "INACTIVE",
					Stackable: true,
					StartsAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					EndsAt:    time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
					Targets: []model.PromotionTarget{{TargetType: "PRODUCT", TargetID: uint64Ptr(1)}},
					Actions: []model.PromotionAction{{ActionType: "PERCENTAGE_DISCOUNT", ValueBasisPoints: intPtr(1000), AppliesTo: "ITEM"}},
			},
		},
	})

	assert.NoError(t, err)
	assert.Len(t, result.AppliedPromotions, 0)
	assert.Len(t, result.SkippedPromotions, 0)
	assert.Equal(t, int64(100000), result.FinalTotal)
}

func TestCalculator_CustomRegisteredAction(t *testing.T) {
	registry := promotion.NewRegistry()
	registry.RegisterAction("LOYALTY_BONUS", func(input promotion.ActionContext) (int64, error) {
		return 3000, nil
	})

	code := "LOYALTY"
	calculator := promotion.NewCalculatorWithRegistry(registry)
	result, err := calculator.Calculate(nil, promotion.CalculationContext{
		Now: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Items: []promotion.CalculationItem{{ProductID: 1, SKU: "P1", ProductName: "P1", CategoryID: 1, Quantity: 1, UnitPrice: 100000}},
		Promotions: []model.Promotion{
			{
				BaseModel: model.BaseModel{ID: 1, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				Code:      &code,
					Name:      "Custom Action",
					Scope:     "CART",
					Status:    "ACTIVE",
					Stackable: true,
					StartsAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					EndsAt:    time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
					Targets:   []model.PromotionTarget{{TargetType: "CART"}},
				Actions:   []model.PromotionAction{{ActionType: "LOYALTY_BONUS", AppliesTo: "CART"}},
			},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(3000), result.DiscountTotal)
	assert.Equal(t, int64(97000), result.FinalTotal)
}

func TestCalculator_NonStackablePromotionBlocksLaterPromotions(t *testing.T) {
	code1 := "ITEM_10"
	code2 := "CART_5"

	calculator := promotion.NewCalculator()
	result, err := calculator.Calculate(nil, promotion.CalculationContext{
		Now: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Items: []promotion.CalculationItem{
			{ProductID: 1, SKU: "P1", ProductName: "P1", CategoryID: 1, Quantity: 1, UnitPrice: 100000},
		},
		Promotions: []model.Promotion{
			{
				BaseModel:  model.BaseModel{ID: 1, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				Code:       &code1,
				Name:       "Item 10%",
				Scope:      "ITEM",
				Status:     "ACTIVE",
				Stackable:  false,
				StartsAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				EndsAt:     time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
				Targets:    []model.PromotionTarget{{TargetType: "PRODUCT", TargetID: uint64Ptr(1)}},
				Actions:    []model.PromotionAction{{ActionType: "PERCENTAGE_DISCOUNT", ValueBasisPoints: intPtr(1000), AppliesTo: "ITEM"}},
			},
			{
				BaseModel: model.BaseModel{ID: 2, CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
				Code:      &code2,
				Name:      "Cart 5%",
				Scope:     "CART",
				Status:    "ACTIVE",
				Stackable: true,
				StartsAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				EndsAt:    time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
				Targets:   []model.PromotionTarget{{TargetType: "CART"}},
				Actions:   []model.PromotionAction{{ActionType: "CART_PERCENTAGE_DISCOUNT", ValueBasisPoints: intPtr(500), AppliesTo: "CART"}},
			},
		},
	})

	assert.NoError(t, err)
	assert.Len(t, result.AppliedPromotions, 1)
	assert.Equal(t, uint64(1), result.AppliedPromotions[0].PromotionID)
	assert.Len(t, result.SkippedPromotions, 1)
	assert.Equal(t, "NON_STACKABLE_ALREADY_APPLIED", result.SkippedPromotions[0].Reason)
}

func TestCalculator_NonStackablePromotionCannotBeAppliedAfterExistingPromotion(t *testing.T) {
	code1 := "ITEM_10"
	code2 := "CART_5"

	calculator := promotion.NewCalculator()
	result, err := calculator.Calculate(nil, promotion.CalculationContext{
		Now: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Items: []promotion.CalculationItem{
			{ProductID: 1, SKU: "P1", ProductName: "P1", CategoryID: 1, Quantity: 1, UnitPrice: 100000},
		},
		Promotions: []model.Promotion{
			{
				BaseModel: model.BaseModel{ID: 1, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				Code:      &code1,
				Name:      "Item 10%",
				Scope:     "ITEM",
				Status:    "ACTIVE",
				Stackable: true,
				StartsAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				EndsAt:    time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
				Targets:   []model.PromotionTarget{{TargetType: "PRODUCT", TargetID: uint64Ptr(1)}},
				Actions:   []model.PromotionAction{{ActionType: "PERCENTAGE_DISCOUNT", ValueBasisPoints: intPtr(1000), AppliesTo: "ITEM"}},
			},
			{
				BaseModel:  model.BaseModel{ID: 2, CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
				Code:       &code2,
				Name:       "Cart 5%",
				Scope:      "CART",
				Status:     "ACTIVE",
				Stackable:  false,
				StartsAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				EndsAt:     time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
				Targets:    []model.PromotionTarget{{TargetType: "CART"}},
				Actions:    []model.PromotionAction{{ActionType: "CART_PERCENTAGE_DISCOUNT", ValueBasisPoints: intPtr(500), AppliesTo: "CART"}},
			},
		},
	})

	assert.NoError(t, err)
	assert.Len(t, result.AppliedPromotions, 1)
	assert.Equal(t, uint64(1), result.AppliedPromotions[0].PromotionID)
	assert.Len(t, result.SkippedPromotions, 1)
	assert.Equal(t, "NON_STACKABLE_CANNOT_STACK", result.SkippedPromotions[0].Reason)
}

func TestCalculator_ExclusivePromotionStopsFurtherProcessing(t *testing.T) {
	code1 := "ITEM_10"
	code2 := "CART_5"

	calculator := promotion.NewCalculator()
	result, err := calculator.Calculate(nil, promotion.CalculationContext{
		Now: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Items: []promotion.CalculationItem{
			{ProductID: 1, SKU: "P1", ProductName: "P1", CategoryID: 1, Quantity: 1, UnitPrice: 100000},
		},
		Promotions: []model.Promotion{
			{
				BaseModel:  model.BaseModel{ID: 1, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				Code:       &code1,
				Name:       "Item 10%",
				Scope:      "ITEM",
				Status:     "ACTIVE",
				Stackable:  true,
				Exclusive:  true,
				StartsAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				EndsAt:     time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
				Targets:    []model.PromotionTarget{{TargetType: "PRODUCT", TargetID: uint64Ptr(1)}},
				Actions:    []model.PromotionAction{{ActionType: "PERCENTAGE_DISCOUNT", ValueBasisPoints: intPtr(1000), AppliesTo: "ITEM"}},
			},
			{
				BaseModel: model.BaseModel{ID: 2, CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
				Code:      &code2,
				Name:      "Cart 5%",
				Scope:     "CART",
				Status:    "ACTIVE",
				Stackable: true,
				StartsAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				EndsAt:    time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
				Targets:   []model.PromotionTarget{{TargetType: "CART"}},
				Actions:   []model.PromotionAction{{ActionType: "CART_PERCENTAGE_DISCOUNT", ValueBasisPoints: intPtr(500), AppliesTo: "CART"}},
			},
		},
	})

	assert.NoError(t, err)
	assert.Len(t, result.AppliedPromotions, 1)
	assert.Equal(t, uint64(1), result.AppliedPromotions[0].PromotionID)
	assert.Len(t, result.SkippedPromotions, 0)
	assert.Contains(t, result.DecisionTrace, "exclusive=true")
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}
