package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/auth"
	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

const (
	ContextUserKey = "auth_user"
)

// RequireAuth valida o JWT enviado no header Authorization e carrega o
// usuário correspondente do banco, disponibilizando-o no contexto da
// requisição. TODA rota autenticada passa por aqui — é a aplicação
// literal da regra "backend valida TODAS as permissões".
//
// Fluxo no sistema: intercepta a requisição antes de qualquer handler;
// se o token for ausente, inválido, expirado, ou o usuário estiver
// inativo, a requisição é encerrada com 401 e o handler nunca é chamado.
func RequireAuth(jwtManager *auth.JWTManager, users *repositories.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token de autenticação ausente"})
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")

		claims, err := jwtManager.Parse(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token inválido ou expirado"})
			return
		}

		user, err := users.FindByID(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "usuário não encontrado"})
			return
		}
		if !user.Active {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "conta desativada"})
			return
		}

		c.Set(ContextUserKey, user)
		c.Next()
	}
}

// RequireRole restringe a rota a um único perfil (ex.: apenas "gestao").
// Deve ser aplicado sempre DEPOIS de RequireAuth na cadeia de middlewares.
// Fluxo no sistema: usado nas rotas administrativas (aprovar contas,
// criar depósitos, gerenciar turmas) — o professor recebe 403 aqui,
// nunca chega a executar lógica de negócio.
func RequireRole(role domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "não autenticado"})
			return
		}
		if user.Role != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "acesso restrito à gestão"})
			return
		}
		c.Next()
	}
}

// CurrentUser extrai o usuário autenticado do contexto da requisição.
// Usado por todo handler que precisa saber "quem" está chamando a API.
func CurrentUser(c *gin.Context) (domain.User, bool) {
	val, exists := c.Get(ContextUserKey)
	if !exists {
		return domain.User{}, false
	}
	user, ok := val.(domain.User)
	return user, ok
}
