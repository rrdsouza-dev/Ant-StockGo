package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/services"
)

// SystemSettingsHandler expõe /settings/retention. Leitura disponível
// para qualquer usuário autenticado (a tela de configurações mostra o
// valor atual mesmo que só a gestão possa alterá-lo); escrita restrita
// à gestão pelo middleware de rota.
type SystemSettingsHandler struct {
	settings *services.SystemSettingsService
}

func NewSystemSettingsHandler(settings *services.SystemSettingsService) *SystemSettingsHandler {
	return &SystemSettingsHandler{settings: settings}
}

// GetRetention — GET /settings/retention
func (h *SystemSettingsHandler) GetRetention(c *gin.Context) {
	settings, err := h.settings.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao carregar configurações"})
		return
	}
	c.JSON(http.StatusOK, settings)
}

type retentionRequest struct {
	MovementRetentionDays int `json:"movement_retention_days"`
}

// UpdateRetention — PUT /settings/retention (somente gestão)
func (h *SystemSettingsHandler) UpdateRetention(c *gin.Context) {
	var req retentionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	updated, err := h.settings.UpdateRetentionDays(req.MovementRetentionDays)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}
