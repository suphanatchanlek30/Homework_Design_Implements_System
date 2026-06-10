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
	CalculationID     string                 `json:"calculationId"`
	OriginalTotal      int64                  `json:"originalTotal"`
	DiscountTotal      int64                  `json:"discountTotal"`
	FinalTotal         int64                  `json:"finalTotal"`
	Currency           string                 `json:"currency"`
	Items              []CalculationItem     `json:"items"`
	AppliedPromotions  []AppliedPromotion     `json:"appliedPromotions"`
	SkippedPromotions  []SkippedPromotion     `json:"skippedPromotions"`
	DecisionTrace      []string               `json:"decisionTrace,omitempty"`
	Snapshot           map[string]any         `json:"snapshot,omitempty"`
}

type Calculator interface {
	Calculate(ctx context.Context, input CalculationContext) (*CalculationResult, error)
}

type calculator struct{}

func NewCalculator() Calculator {
	return &calculator{}
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
		CalculationID:    fmt.Sprintf("calc-%d", input.Now.UnixNano()),
		Currency:         input.Currency,
		Items:            cloneItems(input.Items),
		AppliedPromotions: []AppliedPromotion{},
		SkippedPromotions: []SkippedPromotion{},
		DecisionTrace:    []string{},
		Snapshot:         map[string]any{},
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

	for _, promotion := range input.Promotions {
		if !isPromotionActive(promotion, input.Now) {
			continue
		}

		if promotion.Scope != string(ScopeItem) && promotion.Scope != string(ScopeCart) && promotion.Scope != string(ScopeCoupon) && promotion.Scope != string(ScopeShipping) {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, "INVALID_SCOPE"))
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

		if ok, reason := evaluateConditions(promotion, input, runningCartBase); !ok {
			result.SkippedPromotions = append(result.SkippedPromotions, skipped(promotion, reason))
			continue
		}

		discount := applyPromotion(promotion, result, runningCartBase)
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

		if promotion.ConflictGroup != nil && *promotion.ConflictGroup != "" {
			appliedConflictGroups[*promotion.ConflictGroup] = true
		}

		if promotion.StopProcessing {
			result.DecisionTrace = append(result.DecisionTrace, "stop_processing=true")
			break
		}

		runningCartBase = computeCurrentCartBase(result.Items)
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

func applyPromotion(promotion model.Promotion, result *CalculationResult, currentCartBase int64) int64 {
	matches := matchedItemIndexes(promotion, result.Items)
	discount := int64(0)
	for _, action := range promotion.Actions {
		switch action.ActionType {
		case "PERCENTAGE_DISCOUNT", "CART_PERCENTAGE_DISCOUNT":
			discount += applyPercentage(action, result.Items, matches, promotion.Scope, currentCartBase)
		case "FIXED_AMOUNT_DISCOUNT", "CART_FIXED_AMOUNT_DISCOUNT":
			discount += applyFixed(action, result.Items, matches, promotion.Scope, currentCartBase)
		case "FREE_SHIPPING":
			discount += 0
		}
	}

	if promotion.Scope == string(ScopeItem) {
		discount = applyToItems(result.Items, discount, promotion)
	} else {
		discount = applyToCart(result.Items, discount, currentCartBase)
	}

	return discount
}

func applyPercentage(action model.PromotionAction, items []CalculationItem, matches []int, scope string, cartBase int64) int64 {
	if action.ValueBasisPoints == nil {
		return 0
	}
	base := cartBase
	if scope == string(ScopeItem) {
		base = 0
		for _, index := range matches {
			base += items[index].FinalAmount
		}
	}
	discount := (base * int64(*action.ValueBasisPoints)) / 10000
	if action.MaxDiscountAmount != nil && discount > *action.MaxDiscountAmount {
		discount = *action.MaxDiscountAmount
	}
	return discount
}

func applyFixed(action model.PromotionAction, items []CalculationItem, matches []int, scope string, cartBase int64) int64 {
	if action.ValueAmount == nil {
		return 0
	}
	discount := *action.ValueAmount
	base := cartBase
	if scope == string(ScopeItem) {
		base = 0
		for _, index := range matches {
			base += items[index].FinalAmount
		}
	}
	if discount > base {
		discount = base
	}
	if action.MaxDiscountAmount != nil && discount > *action.MaxDiscountAmount {
		discount = *action.MaxDiscountAmount
	}
	return discount
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

func evaluateConditions(promotion model.Promotion, input CalculationContext, cartBase int64) (bool, string) {
	if len(promotion.Conditions) == 0 {
		return true, ""
	}

	for _, condition := range promotion.Conditions {
		ok, err := evaluateCondition(condition, input, cartBase)
		if !ok {
			return false, err
		}
	}
	return true, ""
}

func evaluateCondition(condition model.PromotionCondition, input CalculationContext, cartBase int64) (bool, string) {
	switch condition.ConditionType {
	case "MIN_ORDER_AMOUNT":
		value := extractInt(condition.ValueJSON)
		if cartBase < value {
			return false, "MIN_ORDER_AMOUNT_NOT_MET"
		}
	case "MAX_ORDER_AMOUNT":
		value := extractInt(condition.ValueJSON)
		if cartBase > value {
			return false, "MAX_ORDER_AMOUNT_EXCEEDED"
		}
	case "COUPON_CODE":
		want := extractString(condition.ValueJSON)
		found := false
		for _, coupon := range input.CouponCodes {
			if strings.EqualFold(coupon, want) {
				found = true
				break
			}
		}
		if !found {
			return false, "COUPON_CODE_MISMATCH"
		}
	case "PAYMENT_METHOD":
		want := extractString(condition.ValueJSON)
		if input.PaymentMethod == nil || !strings.EqualFold(*input.PaymentMethod, want) {
			return false, "PAYMENT_METHOD_MISMATCH"
		}
	case "PRODUCT_ID":
		want := extractUint(condition.ValueJSON)
		found := false
		for _, item := range input.Items {
			if item.ProductID == want {
				found = true
				break
			}
		}
		if !found {
			return false, "PRODUCT_CONDITION_MISMATCH"
		}
	case "CATEGORY_ID":
		want := extractUint(condition.ValueJSON)
		found := false
		for _, item := range input.Items {
			if item.CategoryID == want {
				found = true
				break
			}
		}
		if !found {
			return false, "CATEGORY_CONDITION_MISMATCH"
		}
	}
	_ = condition.Operator
	_ = condition.LogicalOperator
	return true, ""
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

func isPromotionActive(promotion model.Promotion, now time.Time) bool {
	if promotion.Status != "ACTIVE" {
		return false
	}
	return !now.Before(promotion.StartsAt) && !now.After(promotion.EndsAt)
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
