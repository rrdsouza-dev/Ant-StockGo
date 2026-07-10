package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/middleware"
	"wms-backend/internal/services"
)

// DepositHandler expõe /deposits.
//
// GET tem dois modos, escolhidos pelo query param `scope`:
//   - (padrão) entidade administrável: gestão vê todos os depósitos,
//     professor vê os das suas turmas. Usado pela tela "Depósitos" e pelo
//     seletor de depósitos ao vincular uma turma.
//   - `scope=stock`: depósitos cujo ESTOQUE o usuário pode acessar —
//     gestão só o depósito administrativo, professor os da turma "ativa"
//     (`class_id`) ou a união de todas as suas turmas. Usado pelas telas
//     de Estoque, Movimentações, Relatórios, Exportações e Dashboard.
//
// Criação, edição e desativação continuam restritas à gestão pelo
// middleware de rota (gestão de entidade, não de estoque).
type DepositHandler struct {
	deposits *services.DepositService
}

func NewDepositHandler(deposits *services.DepositService) *DepositHandler {
	return &DepositHandler{deposits: deposits}
}

type depositRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	IsAdministrative bool   `json:"is_administrative"`
}

// List — GET /deposits[?scope=stock&class_id=...]
func (h *DepositHandler) List(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)

	if c.Query("scope") == "stock" {
		deposits, err := h.deposits.ListForStock(user, c.Query("class_id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao listar depósitos"})
			return
		}
		c.JSON(http.StatusOK, deposits)
		return
	}

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
	deposit, err := h.deposits.Create(req.Name, req.Description, user.ID, req.IsAdministrative)
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
	deposit, err := h.deposits.Update(id, req.Name, req.Description, req.IsAdministrative)
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
