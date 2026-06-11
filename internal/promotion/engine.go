package promotion

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
)

type ScopeOrder string

const (
	ScopeItem     ScopeOrder = "ITEM"
	ScopeCart     ScopeOrder = "CART"
	ScopeCoupon   ScopeOrder = "COUPON"
	ScopeShipping ScopeOrder = "SHIPPING"
)

type CalculationItem struct {
	ProductID      uint64
	SKU            string
	ProductName    string
	CategoryID     uint64
	Quantity       int
	UnitPrice      int64
	OriginalAmount int64
	DiscountAmount int64
	FinalAmount    int64
}

type CalculationContext struct {
	Now           time.Time
	UserID        *uint64
	Currency      string
	CouponCodes   []string
	PaymentMethod *string
	ShippingMethod *string
	Items         []CalculationItem
	Promotions    []model.Promotion
}

type AppliedPromotion struct {
	PromotionID    uint64 `json:"promotionId"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Scope          string `json:"scope"`
	DiscountAmount int64  `json:"discountAmount"`
}

type SkippedPromotion struct {
	PromotionID uint64 `json:"promotionId"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Reason      string `json:"reason"`
}

type CalculationResult struct {
	CalculationID    string             `json:"calculationId"`
	OriginalTotal    int64              `json:"originalTotal"`
	DiscountTotal    int64              `json:"discountTotal"`
	FinalTotal       int64              `json:"finalTotal"`
	Currency         string             `json:"currency"`
	Items            []CalculationItem  `json:"items"`
	AppliedPromotions []AppliedPromotion `json:"appliedPromotions"`
	SkippedPromotions []SkippedPromotion `json:"skippedPromotions"`
	DecisionTrace    []string           `json:"decisionTrace,omitempty"`
	Snapshot         map[string]any     `json:"snapshot,omitempty"`
}

type Calculator interface {
	Calculate(ctx context.Context, input CalculationContext) (*CalculationResult, error)
}

type calculator struct {
	registry *Registry
}

func NewCalculator() Calculator {
	return &calculator{registry: NewRegistry()}
}

func NewCalculatorWithRegistry(registry *Registry) Calculator {
	if registry == nil {
		registry = NewRegistry()
	}
	return &calculator{registry: registry}
}

func (c *calculator) Calculate(ctx context.Context, input CalculationContext) (*CalculationResult, error) {
	_ = ctx

	if input.Currency == "" {
		input.Currency = "THB"
	}

	sort.SliceStable(input.Promotions, func(i, j int) bool {
		li := scopeRank(input.Promotions[i].Scope)
		lj := scopeRank(input.Promotions[j].Scope)
		if li != lj {
			return li < lj
		}
		if input.Promotions[i].Priority != input.Promotions[j].Priority {
			return input.Promotions[i].Priority < input.Promotions[j].Priority
		}
		if !input.Promotions[i].CreatedAt.Equal(input.Promotions[j].CreatedAt) {
			return input.Promotions[i].CreatedAt.Before(input.Promotions[j].CreatedAt)
		}
		return input.Promotions[i].ID < input.Promotions[j].ID
	})

	result := &CalculationResult{
		CalculationID:     fmt.Sprintf("calc-%d", input.Now.UnixNano()),
		Currency:          input.Currency,
		Items:             cloneItems(input.Items),
		AppliedPromotions: []AppliedPromotion{},
		SkippedPromotions: []SkippedPromotion{},
		DecisionTrace:     []string{},
		Snapshot:          map[string]any{},
	}

	originalTotal := int64(0)
	for i := range result.Items {
		item := &result.Items[i]
		item.OriginalAmount = item.UnitPrice * int64(item.Quantity)
		item.DiscountAmount = 0
		item.FinalAmount = item.OriginalAmount
		originalTotal += item.OriginalAmount
	}

	runningCartBase := originalTotal
	appliedConflictGroups := map[string]bool{}
	hasAppliedPromotions := false
	appliedExclusive := false
	appliedNonStackable := false

	for _, promotion := range input.Promotions {
		if !isPromotionActive(promotion, input.Now) {
			continue
		}
		if !isAllowedScope(promotion.Scope) {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, "INVALID_SCOPE"))
			continue
		}
		if appliedExclusive {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, "EXCLUSIVE_ALREADY_APPLIED"))
			continue
		}
		if promotion.Exclusive && hasAppliedPromotions {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, "EXCLUSIVE_CANNOT_STACK"))
			continue
		}
		if appliedNonStackable {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, "NON_STACKABLE_ALREADY_APPLIED"))
			continue
		}
		if !promotion.Stackable && hasAppliedPromotions {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, "NON_STACKABLE_CANNOT_STACK"))
			continue
		}
		if promotion.ConflictGroup != nil && *promotion.ConflictGroup != "" && appliedConflictGroups[*promotion.ConflictGroup] {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, "CONFLICT_GROUP_BLOCKED"))
			continue
		}
		if !evaluateTargets(promotion, result.Items) {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, "TARGET_MISMATCH"))
			continue
		}
		if ok, reason := evaluateConditions(c.registry, promotion, input, runningCartBase); !ok {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, reason))
			continue
		}

		discount, err := applyPromotion(c.registry, promotion, result, runningCartBase)
		if err != nil {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, err.Error()))
			continue
		}
		if discount <= 0 {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, "NO_DISCOUNT"))
			continue
		}

		result.AppliedPromotions = append(result.AppliedPromotions, AppliedPromotion{
			PromotionID:    promotion.ID,
			Code:           derefString(promotion.Code),
			Name:           promotion.Name,
			Scope:          promotion.Scope,
			DiscountAmount: discount,
		})
		hasAppliedPromotions = true
		if promotion.Exclusive {
			appliedExclusive = true
		}
		if !promotion.Stackable {
			appliedNonStackable = true
		}

		if promotion.ConflictGroup != nil && *promotion.ConflictGroup != "" {
			appliedConflictGroups[*promotion.ConflictGroup] = true
		}

		runningCartBase = computeCurrentCartBase(result.Items)

		if promotion.Exclusive {
			result.DecisionTrace = append(result.DecisionTrace, "exclusive=true")
			break
		}

		if promotion.StopProcessing {
			result.DecisionTrace = append(result.DecisionTrace, "stop_processing=true")
			break
		}
	}

	result.OriginalTotal = originalTotal
	result.DiscountTotal = originalTotal - computeFinalTotal(result.Items)
	result.FinalTotal = computeFinalTotal(result.Items)
	if result.FinalTotal < 0 {
		result.FinalTotal = 0
	}
	result.Snapshot["scopeOrder"] = []string{string(ScopeItem), string(ScopeCart), string(ScopeCoupon), string(ScopeShipping)}
	return result, nil
}

func applyPromotion(registry *Registry, promotion model.Promotion, result *CalculationResult, currentCartBase int64) (int64, error) {
	matches := matchedItemIndexes(promotion, result.Items)
	discount := int64(0)
	for _, action := range promotion.Actions {
		handler, ok := registry.Action(action.ActionType)
		if !ok {
			return 0, fmt.Errorf("ACTION_STRATEGY_NOT_SUPPORTED")
		}
		actionDiscount, err := handler(ActionContext{
			Promotion:      promotion,
			Action:         action,
			Items:          result.Items,
			MatchedIndexes: matches,
			CartBase:       currentCartBase,
		})
		if err != nil {
			return 0, err
		}
		discount += actionDiscount
	}

	if promotion.Scope == string(ScopeItem) {
		discount = applyToItems(result.Items, discount, promotion)
	} else {
		discount = applyToCart(result.Items, discount, currentCartBase)
	}
	return discount, nil
}

func evaluateConditions(registry *Registry, promotion model.Promotion, input CalculationContext, cartBase int64) (bool, string) {
	if len(promotion.Conditions) == 0 {
		return true, ""
	}
	for _, condition := range promotion.Conditions {
		handler, ok := registry.Condition(condition.ConditionType)
		if !ok {
			return false, "CONDITION_STRATEGY_NOT_SUPPORTED"
		}
		okValue, reason, err := handler(ConditionContext{
			Promotion: promotion,
			Condition: condition,
			Input:     input,
			CartBase:  cartBase,
		})
		if err != nil {
			return false, err.Error()
		}
		if !okValue {
			return false, reason
		}
	}
	return true, ""
}

func percentageActionHandler(input ActionContext) (int64, error) {
	if input.Action.ValueBasisPoints == nil {
		return 0, nil
	}
	base := input.CartBase
	if input.Promotion.Scope == string(ScopeItem) {
		base = 0
		for _, index := range input.MatchedIndexes {
			base += input.Items[index].FinalAmount
		}
	}
	discount := (base * int64(*input.Action.ValueBasisPoints)) / 10000
	if input.Action.MaxDiscountAmount != nil && discount > *input.Action.MaxDiscountAmount {
		discount = *input.Action.MaxDiscountAmount
	}
	return discount, nil
}

func fixedAmountActionHandler(input ActionContext) (int64, error) {
	if input.Action.ValueAmount == nil {
		return 0, nil
	}
	discount := *input.Action.ValueAmount
	base := input.CartBase
	if input.Promotion.Scope == string(ScopeItem) {
		base = 0
		for _, index := range input.MatchedIndexes {
			base += input.Items[index].FinalAmount
		}
	}
	if discount > base {
		discount = base
	}
	if input.Action.MaxDiscountAmount != nil && discount > *input.Action.MaxDiscountAmount {
		discount = *input.Action.MaxDiscountAmount
	}
	return discount, nil
}

func freeShippingActionHandler(input ActionContext) (int64, error) {
	return 0, nil
}

func minOrderAmountConditionHandler(input ConditionContext) (bool, string, error) {
	value := extractInt(input.Condition.ValueJSON)
	if input.CartBase < value {
		return false, "MIN_ORDER_AMOUNT_NOT_MET", nil
	}
	return true, "", nil
}

func maxOrderAmountConditionHandler(input ConditionContext) (bool, string, error) {
	value := extractInt(input.Condition.ValueJSON)
	if input.CartBase > value {
		return false, "MAX_ORDER_AMOUNT_EXCEEDED", nil
	}
	return true, "", nil
}

func couponCodeConditionHandler(input ConditionContext) (bool, string, error) {
	want := extractString(input.Condition.ValueJSON)
	for _, coupon := range input.Input.CouponCodes {
		if strings.EqualFold(coupon, want) {
			return true, "", nil
		}
	}
	return false, "COUPON_CODE_MISMATCH", nil
}

func paymentMethodConditionHandler(input ConditionContext) (bool, string, error) {
	want := extractString(input.Condition.ValueJSON)
	if input.Input.PaymentMethod == nil || !strings.EqualFold(*input.Input.PaymentMethod, want) {
		return false, "PAYMENT_METHOD_MISMATCH", nil
	}
	return true, "", nil
}

func productConditionHandler(input ConditionContext) (bool, string, error) {
	want := extractUint(input.Condition.ValueJSON)
	for _, item := range input.Input.Items {
		if item.ProductID == want {
			return true, "", nil
		}
	}
	return false, "PRODUCT_CONDITION_MISMATCH", nil
}

func categoryConditionHandler(input ConditionContext) (bool, string, error) {
	want := extractUint(input.Condition.ValueJSON)
	for _, item := range input.Input.Items {
		if item.CategoryID == want {
			return true, "", nil
		}
	}
	return false, "CATEGORY_CONDITION_MISMATCH", nil
}

func dateRangeConditionHandler(input ConditionContext) (bool, string, error) {
	var window struct {
		StartsAt time.Time `json:"startsAt"`
		EndsAt   time.Time `json:"endsAt"`
	}
	if err := json.Unmarshal(input.Condition.ValueJSON, &window); err != nil {
		return false, "INVALID_DATE_RANGE_CONDITION", err
	}
	if input.Input.Now.Before(window.StartsAt) || input.Input.Now.After(window.EndsAt) {
		return false, "DATE_RANGE_MISMATCH", nil
	}
	return true, "", nil
}

func passthroughConditionHandler(input ConditionContext) (bool, string, error) {
	return true, "", nil
}

func applyToItems(items []CalculationItem, discount int64, promotion model.Promotion) int64 {
	if discount <= 0 || len(items) == 0 {
		return 0
	}
	matches := matchedItemIndexes(promotion, items)
	if len(matches) == 0 {
		return 0
	}

	totalBase := int64(0)
	for _, index := range matches {
		totalBase += items[index].FinalAmount
	}
	if totalBase <= 0 {
		return 0
	}

	remaining := discount
	applied := int64(0)
	for _, index := range matches {
		item := &items[index]
		share := (item.FinalAmount * discount) / totalBase
		if share > item.FinalAmount {
			share = item.FinalAmount
		}
		item.DiscountAmount += share
		item.FinalAmount -= share
		applied += share
		remaining -= share
	}
	if remaining > 0 {
		for _, index := range matches {
			if remaining == 0 {
				break
			}
			item := &items[index]
			if item.FinalAmount <= 0 {
				continue
			}
			adjustment := remaining
			if adjustment > item.FinalAmount {
				adjustment = item.FinalAmount
			}
			item.DiscountAmount += adjustment
			item.FinalAmount -= adjustment
			applied += adjustment
			remaining -= adjustment
		}
	}
	return applied
}

func applyToCart(items []CalculationItem, discount int64, cartBase int64) int64 {
	if discount <= 0 || cartBase <= 0 {
		return 0
	}
	if discount > cartBase {
		discount = cartBase
	}
	remaining := discount
	for i := range items {
		if remaining <= 0 {
			break
		}
		share := (items[i].FinalAmount * discount) / cartBase
		if share > items[i].FinalAmount {
			share = items[i].FinalAmount
		}
		items[i].DiscountAmount += share
		items[i].FinalAmount -= share
		remaining -= share
	}
	if remaining > 0 {
		for i := range items {
			if remaining <= 0 {
				break
			}
			if items[i].FinalAmount <= 0 {
				continue
			}
			adjustment := remaining
			if adjustment > items[i].FinalAmount {
				adjustment = items[i].FinalAmount
			}
			items[i].DiscountAmount += adjustment
			items[i].FinalAmount -= adjustment
			remaining -= adjustment
		}
	}
	return discount - remaining
}

func evaluateTargets(promotion model.Promotion, items []CalculationItem) bool {
	if len(promotion.Targets) == 0 {
		return true
	}
	for _, target := range promotion.Targets {
		switch target.TargetType {
		case "CART":
			return true
		case "PRODUCT":
			if target.TargetID == nil {
				continue
			}
			for _, item := range items {
				if item.ProductID == *target.TargetID {
					return true
				}
			}
		case "CATEGORY":
			if target.TargetID == nil {
				continue
			}
			for _, item := range items {
				if item.CategoryID == *target.TargetID {
					return true
				}
			}
		}
	}
	return false
}

func matchedItemIndexes(promotion model.Promotion, items []CalculationItem) []int {
	indexes := make([]int, 0)
	for i := range items {
		matched := false
		if len(promotion.Targets) == 0 {
			matched = true
		}
		for _, target := range promotion.Targets {
			switch target.TargetType {
			case "CART":
				matched = true
			case "PRODUCT":
				if target.TargetID != nil && items[i].ProductID == *target.TargetID {
					matched = true
				}
			case "CATEGORY":
				if target.TargetID != nil && items[i].CategoryID == *target.TargetID {
					matched = true
				}
			}
			if matched {
				break
			}
		}
		if matched {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func isPromotionActive(promotion model.Promotion, now time.Time) bool {
	if promotion.Status != "ACTIVE" {
		return false
	}
	return !now.Before(promotion.StartsAt) && !now.After(promotion.EndsAt)
}

func isAllowedScope(scope string) bool {
	switch scope {
	case "ITEM", "CART", "COUPON", "SHIPPING":
		return true
	default:
		return false
	}
}

func scopeRank(scope string) int {
	switch scope {
	case string(ScopeItem):
		return 1
	case string(ScopeCart):
		return 2
	case string(ScopeCoupon):
		return 3
	case string(ScopeShipping):
		return 4
	default:
		return 99
	}
}

func cloneItems(items []CalculationItem) []CalculationItem {
	cloned := make([]CalculationItem, len(items))
	copy(cloned, items)
	return cloned
}

func computeCurrentCartBase(items []CalculationItem) int64 {
	total := int64(0)
	for _, item := range items {
		total += item.FinalAmount
	}
	return total
}

func computeFinalTotal(items []CalculationItem) int64 {
	return computeCurrentCartBase(items)
}

func skipped(promotion model.Promotion, reason string) SkippedPromotion {
	return SkippedPromotion{
		PromotionID: promotion.ID,
		Code:        derefString(promotion.Code),
		Name:        promotion.Name,
		Scope:       promotion.Scope,
		Reason:      reason,
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func extractInt(raw []byte) int64 {
	var value int64
	_ = json.Unmarshal(raw, &value)
	return value
}

func extractUint(raw []byte) uint64 {
	var value uint64
	_ = json.Unmarshal(raw, &value)
	return value
}

func extractString(raw []byte) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}
