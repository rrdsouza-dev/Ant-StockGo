package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"wms-backend/config"
	"wms-backend/internal/auth"
	"wms-backend/internal/database"
	"wms-backend/internal/handlers"
	"wms-backend/internal/ia"
	"wms-backend/internal/importing"
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
	categoryRepo := repositories.NewCategoryRepository(db)
	supportRepo := repositories.NewSupportTicketRepository(db)
	preProductRepo := repositories.NewPreProductRepository(db)
	settingsRepo := repositories.NewSystemSettingsRepository(db)

	// Services (regras de negócio)
	classService := services.NewClassService(classRepo, depositRepo)
	depositService := services.NewDepositService(depositRepo, classService)
	inventoryService := services.NewInventoryService(inventoryRepo, depositService)
	authService := services.NewAuthService(userRepo, pendingRepo, jwtManager)
	userService := services.NewUserService(userRepo, classService, depositService)
	categoryService := services.NewCategoryService(categoryRepo)
	supportService := services.NewSupportService(supportRepo, cfg.SupportAdminCode)
	preProductService := services.NewPreProductService(preProductRepo)
	settingsService := services.NewSystemSettingsService(settingsRepo)

	// Otis (assistente de IA) — camada própria em internal/ia, fora de
	// services/, pois não faz acesso a banco e tem ciclo de vida
	// próprio (cliente HTTP para o Ollama). O segundo argumento é o
	// Retriever de RAG: nil nesta V1, pronto para receber uma
	// implementação real (embeddings via pgvector no Supabase) no
	// futuro sem mudar nada aqui além desta linha.
	ollamaClient := ia.NewOllamaClient(cfg.OllamaBaseURL, cfg.OllamaModel, time.Duration(cfg.OllamaTimeoutSecs)*time.Second)
	otisService := ia.NewOtisService(ollamaClient, nil)

	// Importação Inteligente — reaproveita o MESMO cliente Ollama do
	// Otis (não abre uma segunda conexão/configuração). A IA aqui tem um
	// papel bem mais restrito que o do Otis: só resolve ambiguidade de
	// cabeçalhos de planilha (ver internal/importing/ai_analyzer.go);
	// todo o resto do fluxo (parsing, normalização, validação,
	// gravação) é determinístico e não depende do Ollama estar no ar.
	headerResolver := importing.NewOllamaHeaderResolver(ollamaClient)
	importService := importing.NewImportService(inventoryService, preProductService, categoryService, headerResolver)

	// Handlers (tradução HTTP <-> service)
	deps := routes.Dependencies{
		Auth:        handlers.NewAuthHandler(authService, userService),
		Users:       handlers.NewUserHandler(userService),
		Deposits:    handlers.NewDepositHandler(depositService),
		Inventory:   handlers.NewInventoryHandler(inventoryService),
		Classes:     handlers.NewClassHandler(classService),
		Categories:  handlers.NewCategoryHandler(categoryService),
		Support:     handlers.NewSupportHandler(supportService),
		PreProducts: handlers.NewPreProductHandler(preProductService),
		Settings:    handlers.NewSystemSettingsHandler(settingsService),
		Otis:        handlers.NewOtisHandler(otisService),
		Imports:     handlers.NewImportHandler(importService),
		JWTManager:  jwtManager,
		UserRepo:    userRepo,
	}

	// Limpeza automática de movimentações antigas (item novo: reverte a
	// política anterior de "histórico nunca é apagado" — ver
	// SystemSettingsService e RELATORIO.md). Roda uma vez ao subir o
	// servidor e depois a cada hora, sempre lendo o período de retenção
	// configurado no momento (a Gestão pode alterá-lo a qualquer momento
	// pela tela de Configurações, sem precisar reiniciar o backend).
	go func() {
		settingsService.CleanupOldMovements()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			settingsService.CleanupOldMovements()
		}
	}()

	router := gin.Default()
	router.Use(middleware.CORS(cfg.AllowedOrigin))
	routes.Setup(router, deps)

	log.Printf("wms-backend rodando na porta %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("erro ao iniciar servidor: %v", err)
	}
}
