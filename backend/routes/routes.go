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
	Auth       *handlers.AuthHandler
	Users      *handlers.UserHandler
	Deposits   *handlers.DepositHandler
	Inventory  *handlers.InventoryHandler
	Classes    *handlers.ClassHandler
	Categories *handlers.CategoryHandler
	Support    *handlers.SupportHandler

	JWTManager *auth.JWTManager
	UserRepo   *repositories.UserRepository
}

// Setup registra todas as rotas da API sob o prefixo /api/v1, aplicando
// os middlewares de autenticação e autorização exatamente como descrito
// nas regras de negócio.
//
// Importante: criar/editar/excluir/movimentar ITENS DE ESTOQUE deixou de
// ser exclusividade da gestão — qualquer usuário autenticado pode chamar
// essas rotas, e é o InventoryService (via DepositService.CanAccess) quem
// decide, por depósito, se o acesso é permitido. Isso é o que torna
// possível o professor ter CRUD completo dentro das turmas dele, e a
// gestão ficar restrita ao depósito administrativo, sem duplicar a regra
// de autorização aqui nas rotas.
func Setup(router *gin.Engine, deps Dependencies) {
	authRequired := middleware.RequireAuth(deps.JWTManager, deps.UserRepo)
	gestaoOnly := middleware.RequireRole(domain.RoleGestao)
	professorOnly := middleware.RequireRole(domain.RoleProfessor)

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

		// ── Depósitos (estoques) — gestão de ENTIDADE ───────────
		// Criar/editar/excluir o depósito como registro continua
		// restrito à gestão. Acesso ao ESTOQUE de cada depósito é uma
		// decisão totalmente separada, resolvida dentro de /inventory.
		deposits := api.Group("/deposits", authRequired)
		deposits.GET("", deps.Deposits.List)
		deposits.POST("", gestaoOnly, deps.Deposits.Create)
		deposits.PATCH("/:id", gestaoOnly, deps.Deposits.Update)
		deposits.DELETE("/:id", gestaoOnly, deps.Deposits.Delete)

		// ── Inventário e movimentações ──────────────────────────
		// Sem gestaoOnly: o escopo de acesso por depósito (professor na
		// turma dele, gestão só no depósito administrativo) já é
		// aplicado dentro do InventoryService.
		inventory := api.Group("/inventory", authRequired)
		inventory.GET("", deps.Inventory.List)
		inventory.POST("", deps.Inventory.Create)
		inventory.PATCH("/:id", deps.Inventory.Update)
		inventory.DELETE("/:id", deps.Inventory.Delete)
		inventory.POST("/move", deps.Inventory.Move)
		inventory.GET("/movements", deps.Inventory.Movements)

		// ── Turmas ───────────────────────────────────────────────
		classes := api.Group("/classes", authRequired)
		classes.GET("", deps.Classes.List)
		classes.POST("", gestaoOnly, deps.Classes.Create)
		classes.PATCH("/:id", gestaoOnly, deps.Classes.Update)
		classes.DELETE("/:id", gestaoOnly, deps.Classes.Delete)

		// ── Categorias (item de estoque) ─────────────────────────
		// Criação disponível para qualquer usuário autenticado: tanto
		// gestão quanto professor cadastram itens de estoque e ambos
		// podem precisar de uma categoria nova na hora — mesmo
		// handler/service/repository, sem duplicar nada.
		categories := api.Group("/categories", authRequired)
		categories.GET("", deps.Categories.List)
		categories.POST("", deps.Categories.Create)

		// ── Suporte ──────────────────────────────────────────────
		// Abrir chamado é exclusivo do professor; visualizar, exportar
		// (feito no frontend com os dados já retornados) e limpar o
		// histórico são exclusivos da gestão.
		support := api.Group("/support", authRequired)
		support.POST("/tickets", professorOnly, deps.Support.Create)
		support.GET("/tickets", gestaoOnly, deps.Support.List)
		support.DELETE("/tickets", gestaoOnly, deps.Support.ClearAll)
	}
}
