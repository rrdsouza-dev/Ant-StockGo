package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/services"
)

// CategoryHandler expõe /categories. Leitura para qualquer usuário
// autenticado (necessário para exibir a categoria dos itens); criação
// restrita à gestão pelo middleware de rota, já que é dado de catálogo
// administrativo, na mesma linha de depósitos e turmas.
type CategoryHandler struct {
	categories *services.CategoryService
}

func NewCategoryHandler(categories *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{categories: categories}
}

// List — GET /categories
func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.categories.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao listar categorias"})
		return
	}
	c.JSON(http.StatusOK, categories)
}

type categoryRequest struct {
	Name string `json:"name"`
}

// Create — POST /categories (somente gestão)
// Usado pelo botão "+" ao lado do campo Categoria no formulário de item.
func (h *CategoryHandler) Create(c *gin.Context) {
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	category, err := h.categories.Create(req.Name)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrCategoryInUse) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, category)
}
