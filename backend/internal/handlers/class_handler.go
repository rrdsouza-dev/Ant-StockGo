package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/middleware"
	"wms-backend/internal/services"
)

// ClassHandler expõe /classes. Listagem é permitida a qualquer usuário
// autenticado (professor vê só as suas); criação, edição e exclusão são
// restritas à gestão pelo middleware de rota.
type ClassHandler struct {
	classes *services.ClassService
}

func NewClassHandler(classes *services.ClassService) *ClassHandler {
	return &ClassHandler{classes: classes}
}

type classRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TeacherIDs  []string `json:"teacher_ids"`
	DepositIDs  []string `json:"deposit_ids"`
}

// List — GET /classes
func (h *ClassHandler) List(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	classes, err := h.classes.List(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao listar turmas"})
		return
	}
	c.JSON(http.StatusOK, classes)
}

// Create — POST /classes (somente gestão)
func (h *ClassHandler) Create(c *gin.Context) {
	var req classRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	class, err := h.classes.Create(req.Name, req.Description, req.TeacherIDs, req.DepositIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, class)
}

// Update — PATCH /classes/:id (somente gestão)
// Permite atualizar nome/descrição e, quando enviados, os vínculos de
// professores e depósitos (substituição completa da lista).
func (h *ClassHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req classRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	class, err := h.classes.Update(id, req.Name, req.Description, req.TeacherIDs, req.DepositIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, class)
}

// Delete — DELETE /classes/:id (somente gestão)
func (h *ClassHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.classes.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Turma removida."})
}
