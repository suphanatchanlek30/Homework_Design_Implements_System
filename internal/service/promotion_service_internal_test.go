package service

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
)

// TestPromotionCreateColumns_IncludePolicyBooleans verifies that create columns include all policy booleans.
// ตรวจว่า create columns ครอบคลุม field policy แบบ boolean ครบถ้วน
func TestPromotionCreateColumns_IncludePolicyBooleans(t *testing.T) {
	columns := promotionCreateColumns()

	assert.Contains(t, columns, "Stackable")
	assert.Contains(t, columns, "Exclusive")
	assert.Contains(t, columns, "StopProcessing")
}

// TestPromotionPolicyBooleans_DoNotUseGormDefaults ensures policy booleans are written explicitly, not via defaults.
// ตรวจว่า field policy แบบ boolean ไม่พึ่งค่า default ของ GORM แต่ถูกเขียนแบบ explicit
func TestPromotionPolicyBooleans_DoNotUseGormDefaults(t *testing.T) {
	promotionType := reflect.TypeOf(model.Promotion{})

	for _, fieldName := range []string{"Stackable", "Exclusive", "StopProcessing"} {
		field, ok := promotionType.FieldByName(fieldName)

		assert.True(t, ok)
		assert.NotContains(t, strings.ToLower(field.Tag.Get("gorm")), "default:")
	}
}
