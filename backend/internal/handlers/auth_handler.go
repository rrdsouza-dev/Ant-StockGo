package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/domain"
	"wms-backend/internal/services"
)

// AuthHandler expõe /auth/login, /auth/register, /auth/pending e
// /auth/approve. Nenhuma regra de negócio vive aqui — cada método apenas
// decodifica a requisição, chama o AuthService e traduz o resultado
// (ou erro) para uma resposta HTTP.
type AuthHandler struct {
	auth  *services.AuthService
	users *services.UserService
}

func NewAuthHandler(auth *services.AuthService, users *services.UserService) *AuthHandler {
	return &AuthHandler{auth: auth, users: users}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Register — POST /auth/register
// Chama ao criar uma solicitação de conta PENDENTE. Nunca autentica o
// usuário automaticamente: a resposta é apenas uma confirmação de que o
// cadastro foi recebido e aguarda aprovação da gestão.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}

	pending, err := h.auth.Register(req.Name, req.Email, req.Password, domain.Role(req.Role))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Cadastro enviado com sucesso. Aguarde a aprovação da gestão para acessar o sistema.",
		"pending": pending,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login — POST /auth/login
// Retorna o usuário autenticado e um JWT válido por 24h em caso de sucesso.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}

	user, token, err := h.auth.Login(req.Email, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, services.ErrAccountInactive) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Corrigido: antes retornava user.ToPublic() puro, que nunca preenche
	// Classes/Deposits (esses campos só existem em UserService.Me). Isso
	// fazia o frontend iniciar a sessão do professor sem turmas/depósitos
	// até que ele visitasse /profile (o único lugar que chamava /users/me).
	// Agora o login já devolve o usuário totalmente hidratado.
	public, err := h.users.Me(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao carregar dados do usuário"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  public,
		"token": token,
	})
}

// ListPending — GET /auth/pending (somente gestão)
// Lista as contas aguardando aprovação, mais antigas primeiro.
func (h *AuthHandler) ListPending(c *gin.Context) {
	pending, err := h.auth.ListPending()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao listar contas pendentes"})
		return
	}
	c.JSON(http.StatusOK, pending)
}

type approveRequest struct {
	UserID string `json:"user_id"`
	Action string `json:"action"` // "approve" ou "reject"
}

// Approve — POST /auth/approve (somente gestão)
// Aprova (ativa a conta) ou rejeita uma solicitação pendente.
func (h *AuthHandler) Approve(c *gin.Context) {
	var req approveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}
	if req.Action != "approve" && req.Action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action deve ser 'approve' ou 'reject'"})
		return
	}

	user, err := h.auth.ApproveOrReject(req.UserID, req.Action == "approve")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Solicitação rejeitada."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Conta aprovada e ativada.", "user": user.ToPublic()})
}
