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

// NewCalculator builds the default promotion calculator with built-in strategies.
// สร้าง calculator มาตรฐานพร้อม strategy พื้นฐานที่ระบบรองรับมาให้แล้ว
func NewCalculator() Calculator {
	return &calculator{registry: NewRegistry()}
}

// NewCalculatorWithRegistry lets tests or future extensions inject custom strategies.
// เปิดให้ test หรือส่วนขยายในอนาคตส่ง registry แบบกำหนดเองเข้ามาได้
func NewCalculatorWithRegistry(registry *Registry) Calculator {
	if registry == nil {
		registry = NewRegistry()
	}
	return &calculator{registry: registry}
}

// Calculate is the main promotion loop: sort promos, enforce policy, apply discounts, and summarize the result.
// เป็นลูปหลักของการคำนวณ promotion ตั้งแต่เรียงลำดับ ตรวจ policy ลงส่วนลด และสรุปผลลัพธ์
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

// applyPromotion resolves every action in a promotion and then writes the discount back to items or cart totals.
// ประมวลผล action ทุกตัวของ promotion แล้วกระจายส่วนลดกลับไปที่ item หรือทั้ง cart
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

// evaluateConditions checks all configured conditions and returns the first failure reason when a promo should be skipped.
// ตรวจทุก condition ของ promotion และคืนเหตุผลแรกที่ทำให้ต้อง skip ถ้าไม่ผ่าน
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

// percentageActionHandler calculates percentage-based discounts for item or cart scope.
// คำนวณส่วนลดแบบเปอร์เซ็นต์ทั้งกรณี item scope และ cart scope
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

// fixedAmountActionHandler calculates fixed-amount discounts and caps them at the available base.
// คำนวณส่วนลดแบบจำนวนเงินคงที่และกันไม่ให้เกินยอดที่ลดได้จริง
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

// freeShippingActionHandler is a placeholder until shipping cost is modeled separately from item totals.
// ตอนนี้เป็น placeholder เพราะระบบยังไม่ได้แยกต้นทุนค่าส่งออกจากยอดสินค้า
func freeShippingActionHandler(input ActionContext) (int64, error) {
	return 0, nil
}

// minOrderAmountConditionHandler passes only when the current cart base reaches the configured threshold.
// ผ่านได้เมื่อยอดปัจจุบันของตะกร้าถึงขั้นต่ำตามที่ condition กำหนด
func minOrderAmountConditionHandler(input ConditionContext) (bool, string, error) {
	value := extractInt(input.Condition.ValueJSON)
	if input.CartBase < value {
		return false, "MIN_ORDER_AMOUNT_NOT_MET", nil
	}
	return true, "", nil
}

// maxOrderAmountConditionHandler passes only when the current cart base stays below the configured ceiling.
// ผ่านได้เมื่อยอดปัจจุบันของตะกร้ายังไม่เกินเพดานที่ condition กำหนด
func maxOrderAmountConditionHandler(input ConditionContext) (bool, string, error) {
	value := extractInt(input.Condition.ValueJSON)
	if input.CartBase > value {
		return false, "MAX_ORDER_AMOUNT_EXCEEDED", nil
	}
	return true, "", nil
}

// couponCodeConditionHandler checks whether one of the submitted coupon codes matches the promotion rule.
// ตรวจว่ามี coupon code ที่ผู้ใช้ส่งมาตรงกับกติกาของ promotion หรือไม่
func couponCodeConditionHandler(input ConditionContext) (bool, string, error) {
	want := extractString(input.Condition.ValueJSON)
	for _, coupon := range input.Input.CouponCodes {
		if strings.EqualFold(coupon, want) {
			return true, "", nil
		}
	}
	return false, "COUPON_CODE_MISMATCH", nil
}

// paymentMethodConditionHandler restricts a promotion to a specific payment method.
// บังคับให้ promotion ใช้ได้เฉพาะกับ payment method ที่กำหนด
func paymentMethodConditionHandler(input ConditionContext) (bool, string, error) {
	want := extractString(input.Condition.ValueJSON)
	if input.Input.PaymentMethod == nil || !strings.EqualFold(*input.Input.PaymentMethod, want) {
		return false, "PAYMENT_METHOD_MISMATCH", nil
	}
	return true, "", nil
}

// productConditionHandler requires at least one requested product ID to match the condition payload.
// ต้องมีสินค้าอย่างน้อยหนึ่งตัวในคำขอที่ product ID ตรงกับ condition นี้
func productConditionHandler(input ConditionContext) (bool, string, error) {
	want := extractUint(input.Condition.ValueJSON)
	for _, item := range input.Input.Items {
		if item.ProductID == want {
			return true, "", nil
		}
	}
	return false, "PRODUCT_CONDITION_MISMATCH", nil
}

// categoryConditionHandler requires at least one requested category ID to match the condition payload.
// ต้องมีสินค้าอย่างน้อยหนึ่งตัวในคำขอที่ category ID ตรงกับ condition นี้
func categoryConditionHandler(input ConditionContext) (bool, string, error) {
	want := extractUint(input.Condition.ValueJSON)
	for _, item := range input.Input.Items {
		if item.CategoryID == want {
			return true, "", nil
		}
	}
	return false, "CATEGORY_CONDITION_MISMATCH", nil
}

// dateRangeConditionHandler applies a condition-specific time window on top of the promotion start/end dates.
// ตรวจช่วงเวลาเพิ่มเติมของ condition นอกเหนือจากช่วงเวลาเริ่มและจบของ promotion
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

// passthroughConditionHandler keeps future condition types valid in the engine until real logic is added.
// ช่วยให้ condition บางชนิดผ่าน engine ไปก่อนได้จนกว่าจะมี logic จริงมาแทน
func passthroughConditionHandler(input ConditionContext) (bool, string, error) {
	return true, "", nil
}

// applyToItems spreads a discount only across matched items and prevents any item total from going negative.
// กระจายส่วนลดไปเฉพาะ item ที่ match และกันไม่ให้ยอดของ item ใดติดลบ
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

// applyToCart spreads a cart-level discount proportionally across all current item totals.
// กระจายส่วนลดระดับ cart แบบตามสัดส่วนไปยังยอดปัจจุบันของ item ทุกตัว
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

// evaluateTargets answers whether a cart is even eligible to consider a promotion before condition checks run.
// ตรวจว่าตะกร้านี้มีสิทธิ์ถูกพิจารณาโปรนี้หรือไม่ก่อนจะไปเช็ก condition ต่อ
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

// matchedItemIndexes finds which items a promotion should directly discount when it applies at item scope.
// หา index ของ item ที่ promotion ควรลงส่วนลดโดยตรงเมื่อเป็น item scope
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

// isPromotionActive combines status and date window checks used by the runtime loop.
// ตรวจทั้ง status และช่วงเวลาใช้งานว่า promotion ยัง active อยู่หรือไม่
func isPromotionActive(promotion model.Promotion, now time.Time) bool {
	if promotion.Status != "ACTIVE" {
		return false
	}
	return !now.Before(promotion.StartsAt) && !now.After(promotion.EndsAt)
}

// isAllowedScope keeps the runtime strict about the known promotion scopes.
// จำกัดให้ runtime ยอมรับเฉพาะ scope ที่ระบบรู้จักเท่านั้น
func isAllowedScope(scope string) bool {
	switch scope {
	case "ITEM", "CART", "COUPON", "SHIPPING":
		return true
	default:
		return false
	}
}

// scopeRank defines the execution order between ITEM, CART, COUPON, and SHIPPING promotions.
// กำหนดลำดับการทำงานของ promotion ตาม scope เช่น ITEM ก่อน CART
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

// cloneItems makes the calculator mutate a working copy instead of the caller's input slice.
// ทำสำเนา items เพื่อให้ calculator แก้ค่าบนสำเนาแทนข้อมูลต้นฉบับ
func cloneItems(items []CalculationItem) []CalculationItem {
	cloned := make([]CalculationItem, len(items))
	copy(cloned, items)
	return cloned
}

// computeCurrentCartBase sums the running final amounts after each promotion application.
// รวมยอดสุทธิของ item ทั้งหมดหลังจากแต่ละ promotion ถูกนำไปใช้แล้ว
func computeCurrentCartBase(items []CalculationItem) int64 {
	total := int64(0)
	for _, item := range items {
		total += item.FinalAmount
	}
	return total
}

// computeFinalTotal is currently the same as the running cart base because shipping is not modeled separately.
// ตอนนี้ final total เท่ากับ cart base เพราะระบบยังไม่ได้แยกค่าส่งออกมาเป็นอีกก้อน
func computeFinalTotal(items []CalculationItem) int64 {
	return computeCurrentCartBase(items)
}

// skipped standardizes how a skipped promotion is recorded in the response payload.
// สร้างข้อมูล skipped promotion ในรูปแบบมาตรฐานสำหรับ response
func skipped(promotion model.Promotion, reason string) SkippedPromotion {
	return SkippedPromotion{
		PromotionID: promotion.ID,
		Code:        derefString(promotion.Code),
		Name:        promotion.Name,
		Scope:       promotion.Scope,
		Reason:      reason,
	}
}

// derefString safely normalizes nullable promo codes into response-friendly strings.
// แปลง promo code ที่อาจเป็น nil ให้กลายเป็น string ที่ response ใช้งานง่าย
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// extractInt decodes numeric condition payloads stored in JSON.
// แตก JSON ของ condition ที่เป็นตัวเลขจำนวนเต็ม
func extractInt(raw []byte) int64 {
	var value int64
	_ = json.Unmarshal(raw, &value)
	return value
}

// extractUint decodes unsigned numeric condition payloads stored in JSON.
// แตก JSON ของ condition ที่เป็นตัวเลขแบบ unsigned
func extractUint(raw []byte) uint64 {
	var value uint64
	_ = json.Unmarshal(raw, &value)
	return value
}

// extractString decodes string condition payloads stored in JSON.
// แตก JSON ของ condition ที่เก็บเป็นข้อความ
func extractString(raw []byte) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}
