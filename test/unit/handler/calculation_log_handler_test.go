package handler_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/handler"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/service"
)

type mockCalculationLogService struct {
	listFn   func(query dto.CalculationLogQuery) (*dto.CalculationLogListResponse, error)
	getFn    func(calculationID string) (*dto.CalculationLogDetailResponse, error)
	replayFn func(calculationID string, req dto.CalculationLogReplayRequest) (*dto.CalculationLogReplayResponse, error)
}

func (m *mockCalculationLogService) List(_ context.Context, query dto.CalculationLogQuery) (*dto.CalculationLogListResponse, error) {
	return m.listFn(query)
}

func (m *mockCalculationLogService) GetByCalculationID(_ context.Context, calculationID string) (*dto.CalculationLogDetailResponse, error) {
	return m.getFn(calculationID)
}

func (m *mockCalculationLogService) Replay(_ context.Context, calculationID string, req dto.CalculationLogReplayRequest) (*dto.CalculationLogReplayResponse, error) {
	return m.replayFn(calculationID, req)
}

func TestCalculationLogList_InvalidQuery(t *testing.T) {
	app := fiber.New()
	svc := &mockCalculationLogService{
		listFn: func(query dto.CalculationLogQuery) (*dto.CalculationLogListResponse, error) {
			return &dto.CalculationLogListResponse{}, nil
		},
	}

	app.Get("/api/v1/calculation-logs", handler.NewCalculationLogHandler(svc).List)

	req := httptest.NewRequest("GET", "/api/v1/calculation-logs?createdFrom=bad", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "INVALID_QUERY_PARAMETER", body.Error.Code)
}

func TestCalculationLogGetByID_NotFound(t *testing.T) {
	app := fiber.New()
	svc := &mockCalculationLogService{
		getFn: func(calculationID string) (*dto.CalculationLogDetailResponse, error) {
			return nil, service.ErrCalculationLogNotFound
		},
	}

	app.Get("/api/v1/calculation-logs/:calculationId", handler.NewCalculationLogHandler(svc).GetByCalculationID)

	req := httptest.NewRequest("GET", "/api/v1/calculation-logs/calc-404", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "CALCULATION_LOG_NOT_FOUND", body.Error.Code)
}

func TestCalculationLogReplay_UnsupportedMode(t *testing.T) {
	app := fiber.New()
	svc := &mockCalculationLogService{
		replayFn: func(calculationID string, req dto.CalculationLogReplayRequest) (*dto.CalculationLogReplayResponse, error) {
			return nil, service.ErrReplayModeNotSupported
		},
	}

	app.Post("/api/v1/calculation-logs/:calculationId/replay", handler.NewCalculationLogHandler(svc).Replay)

	req := httptest.NewRequest("POST", "/api/v1/calculation-logs/calc-1/replay", strings.NewReader(`{"mode":"CURRENT_CONFIG"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)

	var body handler.ErrorResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "REPLAY_MODE_NOT_SUPPORTED", body.Error.Code)
}

func TestCalculationLogReplay_Success(t *testing.T) {
	app := fiber.New()
	svc := &mockCalculationLogService{
		replayFn: func(calculationID string, req dto.CalculationLogReplayRequest) (*dto.CalculationLogReplayResponse, error) {
			return &dto.CalculationLogReplayResponse{
				CalculationID: calculationID,
				Mode:          "SNAPSHOT_CONFIG",
				Matched:       true,
				Differences:   []string{},
			}, nil
		},
	}

	app.Post("/api/v1/calculation-logs/:calculationId/replay", handler.NewCalculationLogHandler(svc).Replay)

	req := httptest.NewRequest("POST", "/api/v1/calculation-logs/calc-1/replay", strings.NewReader(`{"mode":"SNAPSHOT_CONFIG"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
