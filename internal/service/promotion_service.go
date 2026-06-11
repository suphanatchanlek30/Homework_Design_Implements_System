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

func NewPromotionService(db *gorm.DB, repo repository.PromotionRepository) PromotionService {
	return &promotionService{db: db, repo: repo}
}

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

func createPromotion(tx *gorm.DB, promotion *model.Promotion) *gorm.DB {
	return tx.Select(promotionCreateColumns()).Create(promotion)
}

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

func (s *promotionService) GetByID(ctx context.Context, id uint64) (*dto.PromotionDetailResponse, error) {
	promotion, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPromotionNotFound
	}
	return promotionDetailFromModel(promotion), nil
}

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

func isAllowedScope(scope string) bool {
	switch scope {
	case "ITEM", "CART", "COUPON", "SHIPPING":
		return true
	default:
		return false
	}
}

func isSupportedAction(actionType string) bool {
	for _, supported := range promotion.SupportedActionTypes() {
		if supported == actionType {
			return true
		}
	}
	return false
}

func isSupportedCondition(conditionType string) bool {
	for _, supported := range promotion.SupportedConditionTypes() {
		if supported == conditionType {
			return true
		}
	}
	return false
}

func hasTargetType(targets []dto.PromotionTargetRequest, targetType string) bool {
	for _, target := range targets {
		if target.TargetType == targetType {
			return true
		}
	}
	return false
}

func defaultLogicalOperator(value string) string {
	if value == "" {
		return "AND"
	}
	return value
}

func isDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func derefStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	copyValue := value
	return &copyValue
}

func normalizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func normalizeLimit(limit int) int {
	if limit < 1 || limit > 100 {
		return 10
	}
	return limit
}

func calcTotalPages(total int64, limit int) int {
	if limit <= 0 {
		return 0
	}
	if total == 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}
