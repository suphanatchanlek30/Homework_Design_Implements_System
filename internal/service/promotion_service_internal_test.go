package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromotionCreateColumns_IncludePolicyBooleans(t *testing.T) {
	columns := promotionCreateColumns()

	assert.Contains(t, columns, "Stackable")
	assert.Contains(t, columns, "Exclusive")
	assert.Contains(t, columns, "StopProcessing")
}
