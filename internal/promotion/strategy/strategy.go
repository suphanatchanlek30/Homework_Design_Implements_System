package strategy

import (
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
)

type PromotionAction interface {
	Execute(order *model.Order, action model.PromotionAction) int64
}

type PromotionCondition interface {
	Evaluate(order *model.Order, condition model.PromotionCondition) bool
}
