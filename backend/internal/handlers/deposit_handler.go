package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/middleware"
	"wms-backend/internal/services"
)

// DepositHandler expõe /deposits. Listagem é permitida a qualquer
// usuário autenticado (já filtrada por turma no service); criação,
// edição e desativação são restritas à gestão pelo middleware de rota.
type DepositHandler struct {
	deposits *services.DepositService
}

func NewDepositHandler(deposits *services.DepositService) *DepositHandler {
	return &DepositHandler{deposits: deposits}
}

type depositRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// List — GET /deposits
func (h *DepositHandler) List(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	deposits, err := h.deposits.List(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao listar depósitos"})
		return
	}
	c.JSON(http.StatusOK, deposits)
}

// Create — POST /deposits (somente gestão)
func (h *DepositHandler) Create(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req depositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	deposit, err := h.deposits.Create(req.Name, req.Description, user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, deposit)
}

// Update — PATCH /deposits/:id (somente gestão)
func (h *DepositHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req depositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	deposit, err := h.deposits.Update(id, req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, deposit)
}

// Delete — DELETE /deposits/:id (somente gestão)
// Soft-delete: preserva o histórico de inventário e movimentações.
func (h *DepositHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.deposits.Deactivate(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Depósito desativado."})
}
