package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/domain"
	"wms-backend/internal/middleware"
	"wms-backend/internal/services"
)

// InventoryHandler expõe /inventory (itens de estoque) e /inventory/move
// (registro de entrada/saída). Disponível para professor e gestão — o
// escopo de acesso por depósito é resolvido pelo InventoryService.
type InventoryHandler struct {
	inventory *services.InventoryService
}

func NewInventoryHandler(inventory *services.InventoryService) *InventoryHandler {
	return &InventoryHandler{inventory: inventory}
}

// List — GET /inventory?deposit_id=
func (h *InventoryHandler) List(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	depositID := c.Query("deposit_id")

	items, err := h.inventory.List(user, depositID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

type inventoryItemRequest struct {
	DepositID   string `json:"deposit_id"`
	Name        string `json:"name"`
	SKU         string `json:"sku"`
	MinQuantity int    `json:"min_quantity"`
}

// Create — POST /inventory (somente gestão)
func (h *InventoryHandler) Create(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req inventoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	item, err := h.inventory.Create(user, req.DepositID, req.Name, req.SKU, req.MinQuantity)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

// Update — PATCH /inventory/:id (somente gestão)
func (h *InventoryHandler) Update(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id := c.Param("id")
	var req inventoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	item, err := h.inventory.Update(user, id, req.Name, req.SKU, req.MinQuantity)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// Delete — DELETE /inventory/:id (somente gestão)
func (h *InventoryHandler) Delete(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	id := c.Param("id")
	if err := h.inventory.Deactivate(user, id); err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Item desativado."})
}

type moveRequest struct {
	InventoryItemID string `json:"inventory_item_id"`
	Type            string `json:"type"` // "in" ou "out"
	Quantity        int    `json:"quantity"`
	Note            string `json:"note"`
}

// Move — POST /inventory/move
// Disponível para professor (dentro de seus depósitos) e gestão.
// Toda chamada gera um StockMovement de auditoria.
func (h *InventoryHandler) Move(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req moveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}

	item, movement, err := h.inventory.MoveStock(user, req.InventoryItemID, domain.MovementType(req.Type), req.Quantity, req.Note)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": item, "movement": movement})
}

// Movements — GET /inventory/movements?deposit_id=&limit=
func (h *InventoryHandler) Movements(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	depositID := c.Query("deposit_id")

	movements, err := h.inventory.ListMovements(user, depositID, 200)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, movements)
}

// respondServiceError traduz erros de negócio conhecidos em códigos HTTP
// coerentes, evitando duplicar essa lógica em cada handler.
func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrForbiddenDeposit):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}
