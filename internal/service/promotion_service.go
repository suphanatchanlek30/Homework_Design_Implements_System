package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/model"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/promotion"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrPromotionNotFound             = errors.New("promotion not found")
	ErrPromotionCodeAlreadyExists    = errors.New("promotion code already exists")
	ErrPromotionVersionConflict      = errors.New("promotion version conflict")
	ErrInvalidPromotionConfig        = errors.New("invalid promotion config")
	ErrActionStrategyNotSupported    = errors.New("action strategy not supported")
	ErrTargetRequired                = errors.New("target required")
	ErrFieldNotPatchable             = errors.New("field not patchable")
	ErrPromotionAlreadyInactive      = errors.New("promotion already inactive")
	ErrPromotionAlreadyExpired       = errors.New("promotion already expired")
	ErrPromotionConfigurationInvalid = errors.New("promotion configuration invalid")
)

type PromotionService interface {
	Create(ctx context.Context, req dto.PromotionCreateRequest) (*dto.PromotionSummaryResponse, error)
	List(ctx context.Context, query dto.PromotionListQuery) (*dto.PromotionListResponse, error)
	GetByID(ctx context.Context, id uint64) (*dto.PromotionDetailResponse, error)
	Replace(ctx context.Context, id uint64, req dto.PromotionReplaceRequest) (*dto.PromotionDetailResponse, error)
	Patch(ctx context.Context, id uint64, req dto.PromotionPatchRequest) (*dto.PromotionDetailResponse, error)
	Validate(ctx context.Context, id uint64, req dto.PromotionValidateRequest) (*dto.PromotionValidationResponse, error)
	Activate(ctx context.Context, id uint64, req dto.PromotionActivateRequest) (*dto.PromotionSummaryResponse, error)
	Deactivate(ctx context.Context, id uint64, req dto.PromotionDeactivateRequest) (*dto.PromotionSummaryResponse, error)
	Usages(ctx context.Context, id uint64, query dto.PromotionUsageQuery) (*dto.PromotionUsageResponse, error)
}

type promotionService struct {
	db   *gorm.DB
	repo repository.PromotionRepository
}

// NewPromotionService centralizes promotion lifecycle logic on top of the repository and DB transactions.
// รวม logic วงจรชีวิตของ promotion ไว้บน repository และ transaction ของฐานข้อมูล
func NewPromotionService(db *gorm.DB, repo repository.PromotionRepository) PromotionService {
	return &promotionService{db: db, repo: repo}
}

// Create validates a promotion payload, writes the header row, and then persists its targets, conditions, and actions.
// ตรวจ payload ของ promotion ก่อน แล้วค่อยบันทึกตัวหลักและกติกาย่อยทั้งหมด
func (s *promotionService) Create(ctx context.Context, req dto.PromotionCreateRequest) (*dto.PromotionSummaryResponse, error) {
	if err := validatePromotionBasic(req.Code, req.Scope, req.Priority, req.StartsAt, req.EndsAt); err != nil {
		return nil, err
	}
	if err := validatePromotionConfig(req.Scope, req.Targets, req.Conditions, req.Actions); err != nil {
		return nil, err
	}

	if existing, err := s.repo.FindByCode(ctx, req.Code); err == nil && existing != nil {
		return nil, ErrPromotionCodeAlreadyExists
	}

	promotion := promotionFromCreate(req)
	if promotion.Status == "" {
		promotion.Status = "DRAFT"
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := createPromotion(tx, promotion).Error; err != nil {
			return err
		}
		return persistPromotionRules(tx, promotion.ID, req.Targets, req.Conditions, req.Actions)
	}); err != nil {
		if isDuplicateKey(err) {
			return nil, ErrPromotionCodeAlreadyExists
		}
		return nil, err
	}

	return promotionSummaryFromModel(promotion), nil
}

// createPromotion keeps the insert column list explicit so policy booleans are always written intentionally.
// ระบุคอลัมน์ตอน insert แบบชัดเจนเพื่อให้ field policy ต่าง ๆ ถูกเขียนอย่างตั้งใจเสมอ
func createPromotion(tx *gorm.DB, promotion *model.Promotion) *gorm.DB {
	return tx.Select(promotionCreateColumns()).Create(promotion)
}

// promotionCreateColumns documents which promotion fields are treated as part of the authoritative create payload.
// ระบุว่า field ไหนบ้างถือเป็นส่วนของ payload หลักตอนสร้าง promotion
func promotionCreateColumns() []string {
	return []string{
		"Code",
		"Name",
		"Description",
		"Scope",
		"Priority",
		"Stackable",
		"Exclusive",
		"StopProcessing",
		"ConflictGroup",
		"Status",
		"StartsAt",
		"EndsAt",
		"MaxUsage",
		"MaxUsagePerUser",
		"Version",
	}
}

// List returns paginated promotion summaries with optional filters for status, scope, code, time, and action type.
// คืนรายการ promotion แบบแบ่งหน้าและรองรับการกรองหลายมิติ เช่น status, scope และ action type
func (s *promotionService) List(ctx context.Context, query dto.PromotionListQuery) (*dto.PromotionListResponse, error) {
	page := normalizePage(query.Page)
	limit := normalizeLimit(query.Limit)

	summaries, total, err := s.repo.List(ctx, repository.PromotionListFilter{
		Status:     query.Status,
		Scope:      query.Scope,
		ActionType: query.ActionType,
		Code:       query.Code,
		ActiveAt:   query.ActiveAt,
	}, page, limit, query.Sort)
	if err != nil {
		return nil, err
	}

	items := make([]dto.PromotionSummaryResponse, len(summaries))
	for i, summary := range summaries {
		items[i] = dto.PromotionSummaryResponse{
			PromotionID:    summary.ID,
			Code:           derefString(summary.Code),
			Name:           summary.Name,
			Scope:          summary.Scope,
			Status:         summary.Status,
			Priority:       summary.Priority,
			StartsAt:       summary.StartsAt,
			EndsAt:         summary.EndsAt,
			Version:        summary.Version,
			Stackable:      summary.Stackable,
			Exclusive:      summary.Exclusive,
			StopProcessing: summary.StopProcessing,
			CreatedAt:      summary.CreatedAt,
			UpdatedAt:      summary.UpdatedAt,
		}
	}

	return &dto.PromotionListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: calcTotalPages(total, limit),
		},
	}, nil
}

// GetByID returns the full promotion definition, including targets, conditions, and actions.
// คืน promotion ฉบับเต็มพร้อม targets, conditions และ actions
func (s *promotionService) GetByID(ctx context.Context, id uint64) (*dto.PromotionDetailResponse, error) {
	promotion, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPromotionNotFound
	}
	return promotionDetailFromModel(promotion), nil
}

// Replace performs a full overwrite of a promotion and recreates all nested rule rows in one transaction.
// เขียนทับ promotion ทั้งชุดและสร้างกติกาย่อยใหม่ทั้งหมดภายใน transaction เดียว
func (s *promotionService) Replace(ctx context.Context, id uint64, req dto.PromotionReplaceRequest) (*dto.PromotionDetailResponse, error) {
	if err := validatePromotionBasic(req.Code, req.Scope, req.Priority, req.StartsAt, req.EndsAt); err != nil {
		return nil, err
	}
	if err := validatePromotionConfig(req.Scope, req.Targets, req.Conditions, req.Actions); err != nil {
		return nil, err
	}

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPromotionNotFound
	}
	if current.Version != req.ExpectedVersion {
		return nil, ErrPromotionVersionConflict
	}

	if existing, err := s.repo.FindByCode(ctx, req.Code); err == nil && existing != nil && existing.ID != id {
		return nil, ErrPromotionCodeAlreadyExists
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current.Code = &req.Code
		current.Name = req.Name
		current.Description = derefStringPtr(req.Description)
		current.Scope = req.Scope
		current.Priority = req.Priority
		current.Stackable = req.Stackable
		current.Exclusive = req.Exclusive
		current.StopProcessing = req.StopProcessing
		current.ConflictGroup = req.ConflictGroup
		current.StartsAt = req.StartsAt
		current.EndsAt = req.EndsAt
		current.MaxUsage = req.MaxUsage
		current.MaxUsagePerUser = req.MaxUsagePerUser
		current.Version++

		if err := tx.Save(current).Error; err != nil {
			return err
		}

		if err := tx.Where("promotion_id = ?", current.ID).Delete(&model.PromotionTarget{}).Error; err != nil {
			return err
		}
		if err := tx.Where("promotion_id = ?", current.ID).Delete(&model.PromotionCondition{}).Error; err != nil {
			return err
		}
		if err := tx.Where("promotion_id = ?", current.ID).Delete(&model.PromotionAction{}).Error; err != nil {
			return err
		}
		return persistPromotionRules(tx, current.ID, req.Targets, req.Conditions, req.Actions)
	}); err != nil {
		return nil, err
	}

	updated, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPromotionNotFound
	}
	return promotionDetailFromModel(updated), nil
}

// Patch updates only the editable metadata fields without changing the promotion rules themselves.
// อัปเดตได้เฉพาะ metadata ที่แก้บางส่วนได้โดยไม่แตะ rules ของ promotion
func (s *promotionService) Patch(ctx context.Context, id uint64, req dto.PromotionPatchRequest) (*dto.PromotionDetailResponse, error) {
	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPromotionNotFound
	}
	if current.Version != req.ExpectedVersion {
		return nil, ErrPromotionVersionConflict
	}
	if req.Name == nil && req.Description == nil && req.Priority == nil && req.StartsAt == nil && req.EndsAt == nil {
		return nil, ErrFieldNotPatchable
	}

	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Description != nil {
		current.Description = derefStringPtr(req.Description)
	}
	if req.Priority != nil {
		current.Priority = *req.Priority
	}
	if req.StartsAt != nil {
		current.StartsAt = *req.StartsAt
	}
	if req.EndsAt != nil {
		current.EndsAt = *req.EndsAt
	}
	if !current.StartsAt.IsZero() && !current.EndsAt.IsZero() && !current.StartsAt.Before(current.EndsAt) {
		return nil, ErrInvalidPromotionConfig
	}
	current.Version++

	if err := s.db.WithContext(ctx).Save(current).Error; err != nil {
		return nil, err
	}

	updated, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPromotionNotFound
	}
	return promotionDetailFromModel(updated), nil
}

// Validate checks whether the stored promotion is complete and supported enough to be activated later.
// ตรวจว่า promotion ที่เก็บไว้ครบถ้วนและรองรับพอที่จะเปิดใช้งานต่อได้หรือไม่
func (s *promotionService) Validate(ctx context.Context, id uint64, req dto.PromotionValidateRequest) (*dto.PromotionValidationResponse, error) {
	promotion, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPromotionNotFound
	}
	if promotion.Version != req.ExpectedVersion {
		return nil, ErrPromotionVersionConflict
	}

	errorsList := validatePromotionModel(promotion)
	warnings := make([]string, 0)
	valid := len(errorsList) == 0
	return &dto.PromotionValidationResponse{Valid: valid, Errors: errorsList, Warnings: warnings}, nil
}

// Activate switches a validated draft promotion into ACTIVE state if its validity window still makes sense.
// เปลี่ยน promotion จาก draft เป็น ACTIVE เมื่อผ่าน validation และช่วงเวลายังใช้งานได้
func (s *promotionService) Activate(ctx context.Context, id uint64, req dto.PromotionActivateRequest) (*dto.PromotionSummaryResponse, error) {
	promotion, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPromotionNotFound
	}
	if promotion.Version != req.ExpectedVersion {
		return nil, ErrPromotionVersionConflict
	}
	if errs := validatePromotionModel(promotion); len(errs) > 0 {
		return nil, ErrPromotionConfigurationInvalid
	}
	if !promotion.EndsAt.After(time.Now()) {
		return nil, ErrPromotionAlreadyExpired
	}
	promotion.Status = "ACTIVE"
	promotion.Version++
	if err := s.db.WithContext(ctx).Save(promotion).Error; err != nil {
		return nil, err
	}
	return promotionSummaryFromModel(promotion), nil
}

// Deactivate turns off a promotion without deleting its rule history.
// ปิดการใช้งาน promotion โดยไม่ลบประวัติกติกาที่เคยบันทึกไว้
func (s *promotionService) Deactivate(ctx context.Context, id uint64, req dto.PromotionDeactivateRequest) (*dto.PromotionSummaryResponse, error) {
	promotion, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPromotionNotFound
	}
	if promotion.Version != req.ExpectedVersion {
		return nil, ErrPromotionVersionConflict
	}
	if promotion.Status == "INACTIVE" {
		return nil, ErrPromotionAlreadyInactive
	}
	promotion.Status = "INACTIVE"
	promotion.Version++
	if err := s.db.WithContext(ctx).Save(promotion).Error; err != nil {
		return nil, err
	}
	return promotionSummaryFromModel(promotion), nil
}

// Usages reports how often a promotion has been consumed and how much discount it has given out.
// รายงานว่ามีการใช้ promotion นี้กี่ครั้งและให้ส่วนลดรวมไปเท่าไร
func (s *promotionService) Usages(ctx context.Context, id uint64, query dto.PromotionUsageQuery) (*dto.PromotionUsageResponse, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return nil, ErrPromotionNotFound
	}
	page := normalizePage(query.Page)
	limit := normalizeLimit(query.Limit)

	usages, total, err := s.repo.FindUsages(ctx, id, query.UserID, query.From, query.To, page, limit)
	if err != nil {
		return nil, err
	}

	items := make([]dto.PromotionUsageListItem, len(usages))
	var totalDiscount int64
	for i, usage := range usages {
		totalDiscount += usage.DiscountAmount
		items[i] = dto.PromotionUsageListItem{
			PromotionID:    usage.PromotionID,
			UserID:         usage.UserID,
			OrderID:        usage.OrderID,
			UsageCount:     usage.UsageCount,
			DiscountAmount: usage.DiscountAmount,
			CreatedAt:      usage.CreatedAt,
			UpdatedAt:      usage.UpdatedAt,
		}
	}

	_ = total
	return &dto.PromotionUsageResponse{
		PromotionID:         id,
		TotalUsage:          int64(len(items)),
		TotalDiscountAmount: totalDiscount,
		Items:               items,
	}, nil
}

// validatePromotionBasic rejects malformed promotion headers before any DB work starts.
// กัน promotion header ที่รูปแบบผิดก่อนจะเริ่มทำงานกับฐานข้อมูล
func validatePromotionBasic(code, scope string, priority int, startsAt, endsAt time.Time) error {
	if code == "" || scope == "" {
		return ErrInvalidPromotionConfig
	}
	if priority < 0 {
		return ErrInvalidPromotionConfig
	}
	if !startsAt.Before(endsAt) {
		return ErrInvalidPromotionConfig
	}
	if !isAllowedScope(scope) {
		return ErrInvalidPromotionConfig
	}
	return nil
}

// validatePromotionConfig rejects unsupported strategy types and mismatched target/scope combinations.
// กัน config ที่ใช้ strategy ไม่รองรับหรือจับคู่ target/scope ไม่ถูกต้อง
func validatePromotionConfig(scope string, targets []dto.PromotionTargetRequest, conditions []dto.PromotionConditionRequest, actions []dto.PromotionActionRequest) error {
	if len(actions) == 0 {
		return ErrInvalidPromotionConfig
	}
	if len(targets) == 0 {
		return ErrTargetRequired
	}
	for _, action := range actions {
		if !isSupportedAction(action.ActionType) {
			return ErrActionStrategyNotSupported
		}
	}
	for _, condition := range conditions {
		if !isSupportedCondition(condition.ConditionType) {
			return ErrInvalidPromotionConfig
		}
	}
	if scope == "ITEM" {
		if !hasTargetType(targets, "PRODUCT") && !hasTargetType(targets, "CATEGORY") {
			return ErrTargetRequired
		}
	}
	if scope == "CART" && !hasTargetType(targets, "CART") {
		return ErrTargetRequired
	}
	return nil
}

// validatePromotionModel checks the already-persisted promotion shape used by validate and activate endpoints.
// ตรวจรูปแบบ promotion ที่บันทึกแล้วเพื่อใช้กับ endpoint validate และ activate
func validatePromotionModel(promotion *model.Promotion) []string {
	var errs []string
	if promotion.Code == nil || *promotion.Code == "" {
		errs = append(errs, "code is required")
	}
	if promotion.Name == "" {
		errs = append(errs, "name is required")
	}
	if promotion.Scope == "" {
		errs = append(errs, "scope is required")
	}
	if !promotion.StartsAt.Before(promotion.EndsAt) {
		errs = append(errs, "invalid date range")
	}
	if len(promotion.Actions) == 0 {
		errs = append(errs, "action is required")
	}
	for _, action := range promotion.Actions {
		if !isSupportedAction(action.ActionType) {
			errs = append(errs, fmt.Sprintf("action strategy not supported: %s", action.ActionType))
		}
	}
	for _, condition := range promotion.Conditions {
		if !isSupportedCondition(condition.ConditionType) {
			errs = append(errs, fmt.Sprintf("condition strategy not supported: %s", condition.ConditionType))
		}
	}
	return errs
}

// persistPromotionRules writes the target, condition, and action rows that belong to one promotion.
// บันทึก target, condition และ action ทั้งหมดที่ผูกกับ promotion เดียวกัน
func persistPromotionRules(tx *gorm.DB, promotionID uint64, targets []dto.PromotionTargetRequest, conditions []dto.PromotionConditionRequest, actions []dto.PromotionActionRequest) error {
	for _, target := range targets {
		entity := model.PromotionTarget{
			PromotionID: promotionID,
			TargetType:  target.TargetType,
			TargetID:    target.TargetID,
			TargetValue: target.TargetValue,
		}
		if err := tx.Create(&entity).Error; err != nil {
			return err
		}
	}

	for _, condition := range conditions {
		entity := model.PromotionCondition{
			PromotionID:     promotionID,
			ConditionType:   condition.ConditionType,
			Operator:        condition.Operator,
			ValueJSON:       condition.ValueJSON,
			GroupKey:        condition.GroupKey,
			LogicalOperator: defaultLogicalOperator(condition.LogicalOperator),
		}
		if err := tx.Create(&entity).Error; err != nil {
			return err
		}
	}

	for _, action := range actions {
		entity := model.PromotionAction{
			PromotionID:       promotionID,
			ActionType:        action.ActionType,
			ValueAmount:       action.ValueAmount,
			ValueBasisPoints:  action.ValueBasisPoints,
			ValueJSON:         action.ValueJSON,
			MaxDiscountAmount: action.MaxDiscountAmount,
			AppliesTo:         action.AppliesTo,
		}
		if err := tx.Create(&entity).Error; err != nil {
			return err
		}
	}
	return nil
}

// promotionFromCreate converts an incoming create request into the persisted promotion model.
// แปลงคำขอสร้าง promotion ให้เป็น model ที่พร้อมบันทึกลงฐานข้อมูล
func promotionFromCreate(req dto.PromotionCreateRequest) *model.Promotion {
	code := req.Code
	return &model.Promotion{
		Code:            &code,
		Name:            req.Name,
		Description:     derefStringPtr(req.Description),
		Scope:           req.Scope,
		Priority:        req.Priority,
		Stackable:       req.Stackable,
		Exclusive:       req.Exclusive,
		StopProcessing:  req.StopProcessing,
		ConflictGroup:   req.ConflictGroup,
		Status:          "DRAFT",
		StartsAt:        req.StartsAt,
		EndsAt:          req.EndsAt,
		MaxUsage:        req.MaxUsage,
		MaxUsagePerUser: req.MaxUsagePerUser,
		Version:         1,
	}
}

// promotionSummaryFromModel builds the lightweight summary response from a promotion model.
// แปลง promotion model ให้เป็น response แบบสรุปสำหรับหน้ารายการหรือผลลัพธ์สั้น
func promotionSummaryFromModel(promotion *model.Promotion) *dto.PromotionSummaryResponse {
	return &dto.PromotionSummaryResponse{
		PromotionID:    promotion.ID,
		Code:           derefStringPtr(promotion.Code),
		Name:           promotion.Name,
		Scope:          promotion.Scope,
		Status:         promotion.Status,
		Priority:       promotion.Priority,
		StartsAt:       promotion.StartsAt,
		EndsAt:         promotion.EndsAt,
		Version:        promotion.Version,
		Stackable:      promotion.Stackable,
		Exclusive:      promotion.Exclusive,
		StopProcessing: promotion.StopProcessing,
		CreatedAt:      promotion.CreatedAt,
		UpdatedAt:      promotion.UpdatedAt,
	}
}

// promotionDetailFromModel builds the full detail response including nested rules.
// แปลง promotion model ให้เป็น response แบบเต็มพร้อมกติกาย่อยทั้งหมด
func promotionDetailFromModel(promotion *model.Promotion) *dto.PromotionDetailResponse {
	targets := make([]dto.PromotionTargetRequest, len(promotion.Targets))
	for i, target := range promotion.Targets {
		targets[i] = dto.PromotionTargetRequest{TargetType: target.TargetType, TargetID: target.TargetID, TargetValue: target.TargetValue}
	}

	conditions := make([]dto.PromotionConditionRequest, len(promotion.Conditions))
	for i, condition := range promotion.Conditions {
		conditions[i] = dto.PromotionConditionRequest{
			ConditionType:   condition.ConditionType,
			Operator:        condition.Operator,
			ValueJSON:       json.RawMessage(condition.ValueJSON),
			GroupKey:        condition.GroupKey,
			LogicalOperator: condition.LogicalOperator,
		}
	}

	actions := make([]dto.PromotionActionRequest, len(promotion.Actions))
	for i, action := range promotion.Actions {
		actions[i] = dto.PromotionActionRequest{
			ActionType:        action.ActionType,
			ValueAmount:       action.ValueAmount,
			ValueBasisPoints:  action.ValueBasisPoints,
			ValueJSON:         json.RawMessage(action.ValueJSON),
			MaxDiscountAmount: action.MaxDiscountAmount,
			AppliesTo:         action.AppliesTo,
		}
	}

	return &dto.PromotionDetailResponse{
		PromotionSummaryResponse: *promotionSummaryFromModel(promotion),
		Description:              stringPtr(promotion.Description),
		ConflictGroup:            promotion.ConflictGroup,
		MaxUsage:                 promotion.MaxUsage,
		MaxUsagePerUser:          promotion.MaxUsagePerUser,
		Targets:                  targets,
		Conditions:               conditions,
		Actions:                  actions,
	}
}

// isAllowedScope validates whether a promotion scope is one of the supported values.
// ตรวจว่า scope ของ promotion อยู่ในชุดค่าที่ระบบรองรับหรือไม่
func isAllowedScope(scope string) bool {
	switch scope {
	case "ITEM", "CART", "COUPON", "SHIPPING":
		return true
	default:
		return false
	}
}

// isSupportedAction checks whether an action type is allowed by the current runtime.
// ตรวจว่า action type นี้อยู่ในชุดที่ runtime ปัจจุบันรองรับหรือไม่
func isSupportedAction(actionType string) bool {
	for _, supported := range promotion.SupportedActionTypes() {
		if supported == actionType {
			return true
		}
	}
	return false
}

// isSupportedCondition checks whether a condition type is allowed by the current runtime.
// ตรวจว่า condition type นี้อยู่ในชุดที่ runtime ปัจจุบันรองรับหรือไม่
func isSupportedCondition(conditionType string) bool {
	for _, supported := range promotion.SupportedConditionTypes() {
		if supported == conditionType {
			return true
		}
	}
	return false
}

// hasTargetType answers whether the target list contains at least one target of the requested kind.
// ตรวจว่ารายการ targets มี target ชนิดที่ต้องการอย่างน้อยหนึ่งตัวหรือไม่
func hasTargetType(targets []dto.PromotionTargetRequest, targetType string) bool {
	for _, target := range targets {
		if target.TargetType == targetType {
			return true
		}
	}
	return false
}

// defaultLogicalOperator normalizes empty logical operators to the default AND behavior.
// กำหนดค่า logical operator ว่างให้กลับไปใช้ AND ตามค่าเริ่มต้น
func defaultLogicalOperator(value string) string {
	if value == "" {
		return "AND"
	}
	return value
}

// isDuplicateKey detects MySQL duplicate-key errors so the service can return a domain conflict.
// ตรวจจับ error key ซ้ำจาก MySQL เพื่อแปลงเป็น conflict ระดับ business
func isDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// derefStringPtr safely unwraps a nullable string pointer into a plain string.
// แปลง string pointer ที่อาจเป็น nil ให้กลายเป็น string ปกติอย่างปลอดภัย
func derefStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// derefString safely unwraps a nullable string pointer into a plain string.
// แปลง string pointer ที่อาจเป็น nil ให้กลายเป็น string ปกติอย่างปลอดภัย
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// stringPtr allocates a string pointer for helper use in response mapping and tests.
// สร้าง pointer ของ string เพื่อใช้เป็น helper ในงาน map response หรือ test
func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	copyValue := value
	return &copyValue
}

// normalizePage applies the service-wide default for invalid or missing page values.
// ปรับค่า page ให้กลับไปใช้ค่าเริ่มต้นของ service เมื่อค่าที่ส่งมาไม่ถูกต้อง
func normalizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

// normalizeLimit applies the service-wide default for invalid or missing limit values.
// ปรับค่า limit ให้กลับไปใช้ค่าเริ่มต้นของ service เมื่อค่าที่ส่งมาไม่ถูกต้อง
func normalizeLimit(limit int) int {
	if limit < 1 || limit > 100 {
		return 10
	}
	return limit
}

// calcTotalPages computes the total number of pages from item count and page size.
// คำนวณจำนวนหน้าทั้งหมดจากจำนวนข้อมูลรวมและขนาดต่อหน้า
func calcTotalPages(total int64, limit int) int {
	if limit <= 0 {
		return 0
	}
	if total == 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}
