package dto

type HealthResponse struct {
	Status string `json:"status"`
}

type ReadyDependencies struct {
	MySQL string `json:"mysql"`
	Redis string `json:"redis,omitempty"`
}

type ReadyResponse struct {
	Status       string            `json:"status"`
	Dependencies ReadyDependencies `json:"dependencies"`
}
