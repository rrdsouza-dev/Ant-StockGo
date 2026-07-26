package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/middleware"
	"wms-backend/internal/services"
)

// PreProductHandler expõe /pre-products. Disponível para qualquer
// usuário autenticado (gestão e professor), na mesma linha de
// /categories — ambos são catálogos de apoio ao cadastro de estoque,
// não dados de estoque em si.
type PreProductHandler struct {
	preProducts *services.PreProductService
}

func NewPreProductHandler(preProducts *services.PreProductService) *PreProductHandler {
	return &PreProductHandler{preProducts: preProducts}
}

type preProductRequest struct {
	Name       string  `json:"name"`
	CategoryID *string `json:"category_id"`
	Brand      string  `json:"brand"`
	Unit       string  `json:"unit"`
	Notes      string  `json:"notes"`
}

func (r preProductRequest) toInput() services.PreProductInput {
	return services.PreProductInput{
		Name: r.Name, CategoryID: r.CategoryID, Brand: r.Brand, Unit: r.Unit, Notes: r.Notes,
	}
}

// List — GET /pre-products
func (h *PreProductHandler) List(c *gin.Context) {
	list, err := h.preProducts.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao listar Pré-Produtos"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Create — POST /pre-products
func (h *PreProductHandler) Create(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req preProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	created, err := h.preProducts.Create(user.ID, req.toInput())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// Update — PATCH /pre-products/:id
func (h *PreProductHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req preProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	updated, err := h.preProducts.Update(id, req.toInput())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// Delete — DELETE /pre-products/:id
func (h *PreProductHandler) Delete(c *gin.Context) {
	if err := h.preProducts.Deactivate(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pré-Produto removido."})
}

// FindByBarcode — GET /pre-products/by-barcode/:code
// Usado pelo scanner: ao ler um código que não corresponde a nenhum
// item de estoque, o frontend consulta aqui antes de pedir ao usuário
// para associar manualmente (ver item 6 da especificação).
func (h *PreProductHandler) FindByBarcode(c *gin.Context) {
	found, err := h.preProducts.FindByBarcode(c.Param("code"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "nenhum Pré-Produto associado a este código"})
		return
	}
	c.JSON(http.StatusOK, found)
}

type associateBarcodeRequest struct {
	Barcode string `json:"barcode"`
}

// AssociateBarcode — POST /pre-products/:id/barcode
func (h *PreProductHandler) AssociateBarcode(c *gin.Context) {
	var req associateBarcodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	updated, err := h.preProducts.AssociateBarcode(c.Param("id"), req.Barcode)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrBarcodeInUse) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}
