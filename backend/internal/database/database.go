package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"wms-backend/config"
)

// Connect abre e valida uma conexão com o PostgreSQL (Supabase).
//
// Fluxo no sistema: chamada uma vez em main.go. O *sql.DB retornado é
// injetado em todos os repositories, que são a única camada autorizada
// a executar SQL diretamente.
// Efeitos colaterais: abre um socket de rede com o banco; falha imediata
// (retorna erro) se as credenciais ou o host estiverem incorretos.
func Connect(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir conexão com o banco: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("falha ao conectar ao banco (ping): %w", err)
	}

	return db, nil
}
