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

func NewRegistry() *Registry {
	registry := &Registry{
		actionHandlers:    map[string]ActionHandler{},
		conditionHandlers: map[string]ConditionHandler{},
	}
	registerDefaultActions(registry)
	registerDefaultConditions(registry)
	return registry
}

func (r *Registry) RegisterAction(actionType string, handler ActionHandler) {
	r.actionHandlers[actionType] = handler
}

func (r *Registry) RegisterCondition(conditionType string, handler ConditionHandler) {
	r.conditionHandlers[conditionType] = handler
}

func (r *Registry) Action(actionType string) (ActionHandler, bool) {
	handler, ok := r.actionHandlers[actionType]
	return handler, ok
}

func (r *Registry) Condition(conditionType string) (ConditionHandler, bool) {
	handler, ok := r.conditionHandlers[conditionType]
	return handler, ok
}

func SupportedActionTypes() []string {
	return []string{
		"PERCENTAGE_DISCOUNT",
		"FIXED_AMOUNT_DISCOUNT",
		"CART_PERCENTAGE_DISCOUNT",
		"CART_FIXED_AMOUNT_DISCOUNT",
		"FREE_SHIPPING",
	}
}

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

func registerDefaultActions(registry *Registry) {
	registry.RegisterAction("PERCENTAGE_DISCOUNT", percentageActionHandler)
	registry.RegisterAction("CART_PERCENTAGE_DISCOUNT", percentageActionHandler)
	registry.RegisterAction("FIXED_AMOUNT_DISCOUNT", fixedAmountActionHandler)
	registry.RegisterAction("CART_FIXED_AMOUNT_DISCOUNT", fixedAmountActionHandler)
	registry.RegisterAction("FREE_SHIPPING", freeShippingActionHandler)
}

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

