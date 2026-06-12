package promotion

import "github.com/suphanatchanlek30/homework_design_implements_system/internal/model"

type ActionContext struct {
	Promotion      model.Promotion
	Action         model.PromotionAction
	Items          []CalculationItem
	MatchedIndexes []int
	CartBase       int64
}

type ConditionContext struct {
	Promotion  model.Promotion
	Condition  model.PromotionCondition
	Input      CalculationContext
	CartBase   int64
}

type ActionHandler func(ActionContext) (int64, error)

type ConditionHandler func(ConditionContext) (bool, string, error)

type Registry struct {
	actionHandlers    map[string]ActionHandler
	conditionHandlers map[string]ConditionHandler
}

// NewRegistry creates the runtime registry and installs the built-in action and condition handlers.
// สร้าง registry สำหรับ runtime และลงทะเบียน handler มาตรฐานของ action กับ condition
func NewRegistry() *Registry {
	registry := &Registry{
		actionHandlers:    map[string]ActionHandler{},
		conditionHandlers: map[string]ConditionHandler{},
	}
	registerDefaultActions(registry)
	registerDefaultConditions(registry)
	return registry
}

// RegisterAction adds or replaces the handler for one action type.
// ลงทะเบียนหรือเขียนทับ handler ของ action type หนึ่งรายการ
func (r *Registry) RegisterAction(actionType string, handler ActionHandler) {
	r.actionHandlers[actionType] = handler
}

// RegisterCondition adds or replaces the handler for one condition type.
// ลงทะเบียนหรือเขียนทับ handler ของ condition type หนึ่งรายการ
func (r *Registry) RegisterCondition(conditionType string, handler ConditionHandler) {
	r.conditionHandlers[conditionType] = handler
}

// Action looks up the registered handler for one action type.
// ค้นหา handler ที่ถูกลงทะเบียนไว้สำหรับ action type ที่ระบุ
func (r *Registry) Action(actionType string) (ActionHandler, bool) {
	handler, ok := r.actionHandlers[actionType]
	return handler, ok
}

// Condition looks up the registered handler for one condition type.
// ค้นหา handler ที่ถูกลงทะเบียนไว้สำหรับ condition type ที่ระบุ
func (r *Registry) Condition(conditionType string) (ConditionHandler, bool) {
	handler, ok := r.conditionHandlers[conditionType]
	return handler, ok
}

// SupportedActionTypes returns the action types that promotion validation currently allows.
// คืนรายการ action types ที่ promotion validation ยอมรับใน runtime ปัจจุบัน
func SupportedActionTypes() []string {
	return []string{
		"PERCENTAGE_DISCOUNT",
		"FIXED_AMOUNT_DISCOUNT",
		"CART_PERCENTAGE_DISCOUNT",
		"CART_FIXED_AMOUNT_DISCOUNT",
		"FREE_SHIPPING",
	}
}

// SupportedConditionTypes returns the condition types that promotion validation currently allows.
// คืนรายการ condition types ที่ promotion validation ยอมรับใน runtime ปัจจุบัน
func SupportedConditionTypes() []string {
	return []string{
		"PRODUCT_ID",
		"CATEGORY_ID",
		"MIN_ORDER_AMOUNT",
		"MAX_ORDER_AMOUNT",
		"COUPON_CODE",
		"USER_SEGMENT",
		"FIRST_ORDER",
		"PAYMENT_METHOD",
		"DATE_RANGE",
	}
}

// registerDefaultActions installs the built-in action handlers into one registry instance.
// ลงทะเบียน action handlers มาตรฐานลงใน registry หนึ่งตัว
func registerDefaultActions(registry *Registry) {
	registry.RegisterAction("PERCENTAGE_DISCOUNT", percentageActionHandler)
	registry.RegisterAction("CART_PERCENTAGE_DISCOUNT", percentageActionHandler)
	registry.RegisterAction("FIXED_AMOUNT_DISCOUNT", fixedAmountActionHandler)
	registry.RegisterAction("CART_FIXED_AMOUNT_DISCOUNT", fixedAmountActionHandler)
	registry.RegisterAction("FREE_SHIPPING", freeShippingActionHandler)
}

// registerDefaultConditions installs the built-in condition handlers into one registry instance.
// ลงทะเบียน condition handlers มาตรฐานลงใน registry หนึ่งตัว
func registerDefaultConditions(registry *Registry) {
	registry.RegisterCondition("MIN_ORDER_AMOUNT", minOrderAmountConditionHandler)
	registry.RegisterCondition("MAX_ORDER_AMOUNT", maxOrderAmountConditionHandler)
	registry.RegisterCondition("COUPON_CODE", couponCodeConditionHandler)
	registry.RegisterCondition("PAYMENT_METHOD", paymentMethodConditionHandler)
	registry.RegisterCondition("PRODUCT_ID", productConditionHandler)
	registry.RegisterCondition("CATEGORY_ID", categoryConditionHandler)
	registry.RegisterCondition("DATE_RANGE", dateRangeConditionHandler)
	registry.RegisterCondition("USER_SEGMENT", passthroughConditionHandler)
	registry.RegisterCondition("FIRST_ORDER", passthroughConditionHandler)
}
