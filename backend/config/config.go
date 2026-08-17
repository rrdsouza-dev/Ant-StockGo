package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config concentra toda configuração de ambiente do backend.
// Fluxo no sistema: Load() é chamada uma única vez em main.go, antes
// de qualquer conexão de banco ou servidor HTTP ser iniciada.
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	JWTSecret     string
	JWTExpiryHrs  int
	ServerPort    string
	AllowedOrigin string

	// SupportAdminCode é exigido da gestão para limpar o histórico de
	// chamados de suporte (ver SupportService.ClearHistory). Sem valor
	// configurado, a limpeza fica bloqueada por padrão (nunca aceita
	// código vazio como válido).
	SupportAdminCode string

	// ── Otis / IA (ver internal/ia) ────────────────────────────────
	// OllamaBaseURL nunca é exposto ao frontend — só o backend fala
	// com o Ollama. OllamaTimeoutSecs cobre o tempo de geração do
	// Qwen3 1.7B rodando localmente no VPS.
	OllamaBaseURL     string
	OllamaModel       string
	OllamaTimeoutSecs int
}

// Load lê o arquivo .env (se existir) e as variáveis de ambiente do
// processo, aplicando defaults seguros para desenvolvimento local.
// Efeito colateral: chama log.Fatal se JWT_SECRET não estiver definido,
// pois rodar sem segredo de assinatura é inseguro em qualquer ambiente.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("aviso: nenhum arquivo .env encontrado, usando variáveis de ambiente do sistema")
	}

	cfg := &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "wms"),
		DBSSLMode:  getEnv("DB_SSLMODE", "require"),

		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTExpiryHrs:  24,
		ServerPort:    getEnv("PORT", "8000"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),

		SupportAdminCode: getEnv("SUPPORT_ADMIN_CODE", ""),

		OllamaBaseURL:     getEnv("OLLAMA_BASE_URL", "http://127.0.0.1:11434"),
		OllamaModel:       getEnv("OLLAMA_MODEL", "qwen3:1.7b"),
		OllamaTimeoutSecs: getEnvInt("OLLAMA_TIMEOUT_SECS", 60),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET não definido: configure-o no .env antes de iniciar o servidor")
	}
	if cfg.SupportAdminCode == "" {
		log.Println("aviso: SUPPORT_ADMIN_CODE não definido — a limpeza do histórico de suporte ficará sempre bloqueada até configurá-lo")
	}

	return cfg
}

// DSN monta a connection string do PostgreSQL (compatível com Supabase).
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
