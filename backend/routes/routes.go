package routes

import (
	"github.com/gin-gonic/gin"
	"wms-backend/internal/auth"
	"wms-backend/internal/domain"
	"wms-backend/internal/handlers"
	"wms-backend/internal/middleware"
	"wms-backend/internal/repositories"
)

// Dependencies agrupa todos os handlers já construídos em main.go.
// Mantém Setup enxuto e fácil de ler — cada linha abaixo mapeia
// diretamente para um item da lista de ENDPOINTS da especificação.
type Dependencies struct {
	Auth      *handlers.AuthHandler
	Users     *handlers.UserHandler
	Deposits  *handlers.DepositHandler
	Inventory *handlers.InventoryHandler
	Classes   *handlers.ClassHandler

	JWTManager *auth.JWTManager
	UserRepo   *repositories.UserRepository
}

// Setup registra todas as rotas da API sob o prefixo /api/v1, aplicando
// os middlewares de autenticação e autorização exatamente como descrito
// nas regras de negócio: professor tem acesso de leitura/movimentação,
// gestão tem acesso total.
func Setup(router *gin.Engine, deps Dependencies) {
	authRequired := middleware.RequireAuth(deps.JWTManager, deps.UserRepo)
	gestaoOnly := middleware.RequireRole(domain.RoleGestao)

	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	api := router.Group("/api/v1")
	{
		// ── Autenticação (públicas) ─────────────────────────────
		authGroup := api.Group("/auth")
		authGroup.POST("/login", deps.Auth.Login)
		authGroup.POST("/register", deps.Auth.Register)

		// ── Autenticação (somente gestão) ───────────────────────
		authGroup.GET("/pending", authRequired, gestaoOnly, deps.Auth.ListPending)
		authGroup.POST("/approve", authRequired, gestaoOnly, deps.Auth.Approve)

		// ── Usuários ─────────────────────────────────────────────
		users := api.Group("/users", authRequired)
		users.GET("/me", deps.Users.Me)
		users.GET("", gestaoOnly, deps.Users.List)

		// ── Depósitos (estoques) ────────────────────────────────
		deposits := api.Group("/deposits", authRequired)
		deposits.GET("", deps.Deposits.List)
		deposits.POST("", gestaoOnly, deps.Deposits.Create)
		deposits.PATCH("/:id", gestaoOnly, deps.Deposits.Update)
		deposits.DELETE("/:id", gestaoOnly, deps.Deposits.Delete)

		// ── Inventário e movimentações ──────────────────────────
		inventory := api.Group("/inventory", authRequired)
		inventory.GET("", deps.Inventory.List)
		inventory.POST("", gestaoOnly, deps.Inventory.Create)
		inventory.PATCH("/:id", gestaoOnly, deps.Inventory.Update)
		inventory.DELETE("/:id", gestaoOnly, deps.Inventory.Delete)
		inventory.POST("/move", deps.Inventory.Move)
		inventory.GET("/movements", deps.Inventory.Movements)

		// ── Turmas ───────────────────────────────────────────────
		classes := api.Group("/classes", authRequired)
		classes.GET("", deps.Classes.List)
		classes.POST("", gestaoOnly, deps.Classes.Create)
		classes.PATCH("/:id", gestaoOnly, deps.Classes.Update)
		classes.DELETE("/:id", gestaoOnly, deps.Classes.Delete)
	}
}
