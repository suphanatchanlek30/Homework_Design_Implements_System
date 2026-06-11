package model

import (
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type ProductCategory struct {
	BaseModel
	Name     string           `gorm:"size:255;not null" json:"name"`
	ParentID *uint64          `json:"parent_id"`
	Status   string           `gorm:"type:enum('ACTIVE','INACTIVE');default:'ACTIVE';not null" json:"status"`
	Parent   *ProductCategory `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Products []Product        `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

type Product struct {
	BaseModel
	SKU         string          `gorm:"size:100;not null;uniqueIndex" json:"sku"`
	Name        string          `gorm:"size:255;not null" json:"name"`
	CategoryID  uint64          `gorm:"not null" json:"category_id"`
	PriceAmount int64           `gorm:"not null" json:"price_amount"`
	Currency    string          `gorm:"size:10;default:'THB';not null" json:"currency"`
	Status      string          `gorm:"type:enum('ACTIVE','INACTIVE');default:'ACTIVE';not null" json:"status"`
	Category    ProductCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

type Order struct {
	BaseModel
	OrderNo                 string      `gorm:"size:100;not null;uniqueIndex" json:"order_no"`
	IdempotencyKey          string      `gorm:"size:120;not null;uniqueIndex" json:"-"`
	RequestHash             string      `gorm:"size:128;not null;index" json:"-"`
	CalculationID           string      `gorm:"size:100;not null;index" json:"calculation_id"`
	UserID                  *uint64     `gorm:"index" json:"user_id,omitempty"`
	OriginalTotal           int64       `gorm:"default:0;not null" json:"original_total"`
	DiscountTotal           int64       `gorm:"default:0;not null" json:"discount_total"`
	FinalTotal              int64       `gorm:"default:0;not null" json:"final_total"`
	Currency                string      `gorm:"size:10;default:'THB';not null" json:"currency"`
	Status                  string      `gorm:"type:enum('DRAFT','CONFIRMED','PAID','CANCELLED');default:'DRAFT';not null" json:"status"`
	AppliedPromotionsJSON   []byte      `gorm:"type:json;not null" json:"-"`
	SkippedPromotionsJSON   []byte      `gorm:"type:json;not null" json:"-"`
	CalculationSnapshotJSON []byte      `gorm:"type:json;not null" json:"-"`
	Items                   []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

type OrderItem struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID        uint64    `gorm:"not null;index" json:"order_id"`
	ProductID      uint64    `gorm:"not null;index" json:"product_id"`
	ProductName    string    `gorm:"size:255;not null" json:"product_name"`
	SKU            string    `gorm:"size:100;not null" json:"sku"`
	Quantity       int       `gorm:"not null" json:"quantity"`
	UnitPrice      int64     `gorm:"not null" json:"unit_price"`
	OriginalAmount int64     `gorm:"not null" json:"original_amount"`
	DiscountAmount int64     `gorm:"default:0;not null" json:"discount_amount"`
	FinalAmount    int64     `gorm:"not null" json:"final_amount"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Promotion struct {
	BaseModel
	Code            *string              `gorm:"size:100;uniqueIndex" json:"code"`
	Name            string               `gorm:"size:255;not null" json:"name"`
	Description     string               `gorm:"type:text" json:"description"`
	Scope           string               `gorm:"type:enum('ITEM','CART','COUPON','SHIPPING');not null" json:"scope"`
	Priority        int                  `gorm:"default:100;not null" json:"priority"`
	Stackable       bool                 `gorm:"not null" json:"stackable"`
	Exclusive       bool                 `gorm:"not null" json:"exclusive"`
	StopProcessing  bool                 `gorm:"not null" json:"stop_processing"`
	ConflictGroup   *string              `gorm:"size:100" json:"conflict_group"`
	Status          string               `gorm:"type:enum('DRAFT','ACTIVE','INACTIVE','EXPIRED');default:'DRAFT';not null" json:"status"`
	StartsAt        time.Time            `gorm:"not null" json:"starts_at"`
	EndsAt          time.Time            `gorm:"not null" json:"ends_at"`
	MaxUsage        *int                 `json:"max_usage"`
	MaxUsagePerUser *int                 `json:"max_usage_per_user"`
	Version         int                  `gorm:"default:1;not null" json:"version"`
	Targets         []PromotionTarget    `gorm:"foreignKey:PromotionID" json:"targets,omitempty"`
	Conditions      []PromotionCondition `gorm:"foreignKey:PromotionID" json:"conditions,omitempty"`
	Actions         []PromotionAction    `gorm:"foreignKey:PromotionID" json:"actions,omitempty"`
}

type PromotionTarget struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	PromotionID uint64    `gorm:"not null;index" json:"promotion_id"`
	TargetType  string    `gorm:"type:enum('PRODUCT','CATEGORY','CART','USER_SEGMENT','BRAND');not null" json:"target_type"`
	TargetID    *uint64   `json:"target_id"`
	TargetValue *string   `gorm:"size:255" json:"target_value"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PromotionCondition struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	PromotionID     uint64    `gorm:"not null;index" json:"promotion_id"`
	ConditionType   string    `gorm:"type:enum('PRODUCT_ID','CATEGORY_ID','MIN_ORDER_AMOUNT','MAX_ORDER_AMOUNT','COUPON_CODE','USER_SEGMENT','FIRST_ORDER','PAYMENT_METHOD','DATE_RANGE');not null" json:"condition_type"`
	Operator        string    `gorm:"type:enum('EQ','NEQ','IN','NOT_IN','GTE','LTE','BETWEEN');not null" json:"operator"`
	ValueJSON       []byte    `gorm:"type:json;not null" json:"value_json"`
	GroupKey        *string   `gorm:"size:50" json:"group_key"`
	LogicalOperator string    `gorm:"type:enum('AND','OR');default:'AND';not null" json:"logical_operator"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PromotionAction struct {
	ID                uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	PromotionID       uint64    `gorm:"not null;index" json:"promotion_id"`
	ActionType        string    `gorm:"type:enum('PERCENTAGE_DISCOUNT','FIXED_AMOUNT_DISCOUNT','CART_PERCENTAGE_DISCOUNT','CART_FIXED_AMOUNT_DISCOUNT','FREE_SHIPPING','BUY_X_GET_Y','BUNDLE_DISCOUNT');not null" json:"action_type"`
	ValueAmount       *int64    `json:"value_amount"`
	ValueBasisPoints  *int      `json:"value_basis_points"`
	ValueJSON         []byte    `gorm:"type:json" json:"value_json"`
	MaxDiscountAmount *int64    `json:"max_discount_amount"`
	AppliesTo         string    `gorm:"type:enum('ITEM','CART','SHIPPING');not null" json:"applies_to"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type PromotionUsage struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	PromotionID    uint64    `gorm:"not null;index" json:"promotion_id"`
	UserID         *uint64   `json:"user_id"`
	OrderID        *uint64   `json:"order_id"`
	UsageCount     int       `gorm:"default:1;not null" json:"usage_count"`
	DiscountAmount int64     `gorm:"default:0;not null" json:"discount_amount"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PromotionCalculationLog struct {
	ID                      uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CalculationID           string    `gorm:"size:100;not null;uniqueIndex" json:"calculation_id"`
	OrderID                 *uint64   `json:"order_id"`
	RequestID               string    `gorm:"size:100;not null;index" json:"request_id"`
	UserID                  *uint64   `json:"user_id"`
	OriginalTotal           int64     `gorm:"not null" json:"original_total"`
	DiscountTotal           int64     `gorm:"not null" json:"discount_total"`
	FinalTotal              int64     `gorm:"not null" json:"final_total"`
	AppliedPromotionsJSON   []byte    `gorm:"type:json;not null" json:"applied_promotions_json"`
	SkippedPromotionsJSON   []byte    `gorm:"type:json;not null" json:"skipped_promotions_json"`
	CalculationSnapshotJSON []byte    `gorm:"type:json;not null" json:"calculation_snapshot_json"`
	CreatedAt               time.Time `json:"created_at"`
}
