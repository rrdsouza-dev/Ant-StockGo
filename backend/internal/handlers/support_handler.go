package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/domain"
	"wms-backend/internal/middleware"
	"wms-backend/internal/services"
)

// SupportHandler expõe /support/tickets. Criação é exclusiva do
// professor; listagem e limpeza de histórico são exclusivas da gestão
// (ambas restritas pelo middleware de rota).
type SupportHandler struct {
	support *services.SupportService
}

func NewSupportHandler(support *services.SupportService) *SupportHandler {
	return &SupportHandler{support: support}
}

type ticketRequest struct {
	Categoria string `json:"categoria"`
	Tipo      string `json:"tipo"`
	Codigo    string `json:"codigo"`
	Descricao string `json:"descricao"`
}

// Create — POST /support/tickets (somente professor)
// Nome e e-mail nunca vêm do corpo da requisição — sempre do usuário
// autenticado, conforme a especificação ("somente leitura, vindos do backend").
func (h *SupportHandler) Create(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req ticketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}

	ticket, err := h.support.Create(user, services.TicketInput{
		Categoria: domain.SupportCategory(req.Categoria),
		Tipo:      domain.SupportIssueType(req.Tipo),
		Codigo:    req.Codigo,
		Descricao: req.Descricao,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ticket)
}

// List — GET /support/tickets (somente gestão)
func (h *SupportHandler) List(c *gin.Context) {
	tickets, err := h.support.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao listar chamados"})
		return
	}
	c.JSON(http.StatusOK, tickets)
}

type clearHistoryRequest struct {
	AdminCode string `json:"admin_code"`
}

// ClearAll — DELETE /support/tickets (somente gestão)
// Exige o código administrativo, validado exclusivamente no backend.
func (h *SupportHandler) ClearAll(c *gin.Context) {
	var req clearHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	if err := h.support.ClearHistory(req.AdminCode); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrInvalidAdminCode) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Histórico de chamados apagado."})
}
