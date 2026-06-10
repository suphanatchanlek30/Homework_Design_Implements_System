package unit

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/dto"
	"github.com/suphanatchanlek30/homework_design_implements_system/internal/handler"
)

func TestHealthz(t *testing.T) {
	app := fiber.New()
	h := handler.NewHealthHandler(nil) // DB not needed for Healthz

	app.Get("/api/v1/healthz", h.Healthz)

	req := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)

	var res dto.HealthResponse
	json.NewDecoder(resp.Body).Decode(&res)
	assert.Equal(t, "UP", res.Status)
}

func TestReadyz_NotReady(t *testing.T) {
	app := fiber.New()
	h := handler.NewHealthHandler(nil) // Nil DB should trigger NOT_READY

	app.Get("/api/v1/readyz", h.Readyz)

	req := httptest.NewRequest("GET", "/api/v1/readyz", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, 503, resp.StatusCode)

	var res dto.ReadyResponse
	json.NewDecoder(resp.Body).Decode(&res)
	assert.Equal(t, "NOT_READY", res.Status)
	assert.Equal(t, "DOWN", res.Dependencies.MySQL)
}
