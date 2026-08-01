package handlers

import (
	"errors"
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

// Delete — DELETE /users/:id (somente gestão)
// Exclui permanentemente uma conta de professor. O service recusa
// qualquer tentativa de excluir uma conta que não seja de professor.
func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.users.DeleteProfessor(c.Param("id")); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrCannotDeleteNonProfessor) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Conta de professor excluída."})
}
