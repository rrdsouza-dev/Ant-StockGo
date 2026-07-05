package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"wms-backend/config"
	"wms-backend/internal/auth"
	"wms-backend/internal/database"
	"wms-backend/internal/handlers"
	"wms-backend/internal/middleware"
	"wms-backend/internal/repositories"
	"wms-backend/internal/services"
	"wms-backend/routes"
)

// main é o único ponto de composição do sistema: carrega config, abre o
// banco, monta a cadeia repository -> service -> handler -> rota, e sobe
// o servidor HTTP. Nenhuma outra parte do código instancia repositories
// ou services diretamente — tudo nasce aqui e é injetado por construtor.
func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("erro ao conectar ao banco: %v", err)
	}
	defer db.Close()
	log.Println("conectado ao PostgreSQL/Supabase com sucesso")

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiryHrs)

	// Repositories (única camada que executa SQL)
	userRepo := repositories.NewUserRepository(db)
	pendingRepo := repositories.NewPendingUserRepository(db)
	depositRepo := repositories.NewDepositRepository(db)
	inventoryRepo := repositories.NewInventoryRepository(db)
	classRepo := repositories.NewClassRepository(db)

	// Services (regras de negócio)
	classService := services.NewClassService(classRepo, depositRepo)
	depositService := services.NewDepositService(depositRepo, classService)
	inventoryService := services.NewInventoryService(inventoryRepo, depositService)
	authService := services.NewAuthService(userRepo, pendingRepo, jwtManager)
	userService := services.NewUserService(userRepo, classService, depositService)

	// Handlers (tradução HTTP <-> service)
	deps := routes.Dependencies{
		Auth:       handlers.NewAuthHandler(authService),
		Users:      handlers.NewUserHandler(userService),
		Deposits:   handlers.NewDepositHandler(depositService),
		Inventory:  handlers.NewInventoryHandler(inventoryService),
		Classes:    handlers.NewClassHandler(classService),
		JWTManager: jwtManager,
		UserRepo:   userRepo,
	}

	router := gin.Default()
	router.Use(middleware.CORS(cfg.AllowedOrigin))
	routes.Setup(router, deps)

	log.Printf("wms-backend rodando na porta %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("erro ao iniciar servidor: %v", err)
	}
}
