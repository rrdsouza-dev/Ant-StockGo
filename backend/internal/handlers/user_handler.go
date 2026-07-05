package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/middleware"
	"wms-backend/internal/services"
)

// UserHandler expõe /users/me (qualquer usuário autenticado) e
// /users (somente gestão).
type UserHandler struct {
	users *services.UserService
}

func NewUserHandler(users *services.UserService) *UserHandler {
	return &UserHandler{users: users}
}

// Me — GET /users/me
// Retorna o perfil do usuário autenticado com turmas e depósitos
// vinculados, usado pelo frontend para montar a sidebar e o seletor
// de depósito assim que a sessão é iniciada.
func (h *UserHandler) Me(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "não autenticado"})
		return
	}
	public, err := h.users.Me(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao carregar perfil"})
		return
	}
	c.JSON(http.StatusOK, public)
}

// List — GET /users (somente gestão)
// Lista todas as contas ativas do sistema (painel administrativo).
func (h *UserHandler) List(c *gin.Context) {
	users, err := h.users.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao listar usuários"})
		return
	}
	public := make([]any, 0, len(users))
	for _, u := range users {
		public = append(public, u.ToPublic())
	}
	c.JSON(http.StatusOK, public)
}
