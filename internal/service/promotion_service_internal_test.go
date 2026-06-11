package service

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
)

func TestPromotionCreateColumns_IncludePolicyBooleans(t *testing.T) {
	columns := promotionCreateColumns()

	assert.Contains(t, columns, "Stackable")
	assert.Contains(t, columns, "Exclusive")
	assert.Contains(t, columns, "StopProcessing")
}

func TestPromotionPolicyBooleans_DoNotUseGormDefaults(t *testing.T) {
	promotionType := reflect.TypeOf(model.Promotion{})

	for _, fieldName := range []string{"Stackable", "Exclusive", "StopProcessing"} {
		field, ok := promotionType.FieldByName(fieldName)

		assert.True(t, ok)
		assert.NotContains(t, strings.ToLower(field.Tag.Get("gorm")), "default:")
	}
}
