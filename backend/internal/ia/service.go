package ia

import (
	"context"
	"errors"
	"strings"
)

var (
	// ErrEmptyMessage é retornado quando o usuário envia uma mensagem em
	// branco. O handler traduz isso para 400.
	ErrEmptyMessage = errors.New("mensagem não pode estar vazia")

	// ErrMessageTooLong protege o Ollama (e o modelo pequeno, 1.7B) de
	// receber prompts desproporcionais vindos de um único turno.
	ErrMessageTooLong = errors.New("mensagem muito longa")

	// ErrTooManyHistoryMessages limita o histórico enviado a cada turno.
	// A V1 não persiste conversa (ela vive só no frontend), mas o
	// backend nunca deve confiar em um histórico de tamanho arbitrário
	// vindo do cliente.
	ErrTooManyHistoryMessages = errors.New("histórico de conversa muito longo")
)

const (
	maxMessageLength = 2000
	maxHistoryTurns  = 20
)

// Message representa um turno da conversa (usuário ou assistente).
// Mesmo formato usado internamente pelo OllamaClient — o frontend envia
// só isto, sem metadados adicionais.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Retriever é o contrato que um futuro RetrievalService (RAG sobre a
// documentação do ANT-Stock, com embeddings em pgvector no Supabase)
// precisará implementar. Não é usado nesta V1 — OtisService funciona
// normalmente com Retriever == nil — mas já existe para que ligar o RAG
// no futuro seja só passar uma implementação real aqui, sem tocar no
// handler nem no contrato HTTP do chat.
type Retriever interface {
	Retrieve(ctx context.Context, query string) (string, error)
}

// OtisService concentra a orquestração do assistente Otis: valida a
// entrada, monta o prompt (com contexto de RAG quando disponível) e
// delega a chamada ao modelo para o OllamaClient.
//
// Segue o mesmo formato de todo *Service do projeto: dependências
// injetadas por construtor, nenhuma lógica de transporte HTTP aqui
// (isso é responsabilidade do OtisHandler).
type OtisService struct {
	ollama    *OllamaClient
	retriever Retriever // nil nesta V1; preparado para RAG futuro
}

// NewOtisService cria o service. retriever pode ser nil — quando for,
// o Otis responde sem nenhum contexto adicional de documentação.
func NewOtisService(ollama *OllamaClient, retriever Retriever) *OtisService {
	return &OtisService{ollama: ollama, retriever: retriever}
}

// Ask valida a mensagem e o histórico recebidos, opcionalmente recupera
// contexto (RAG, quando o retriever existir) e pergunta ao modelo.
//
// Fluxo no sistema: chamado exclusivamente por OtisHandler.Chat, já
// depois da autenticação (RequireAuth) ter sido aplicada na rota.
func (s *OtisService) Ask(ctx context.Context, message string, history []Message) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", ErrEmptyMessage
	}
	if len(message) > maxMessageLength {
		return "", ErrMessageTooLong
	}
	if len(history) > maxHistoryTurns {
		return "", ErrTooManyHistoryMessages
	}

	retrievedContext := ""
	if s.retriever != nil {
		// Erros de recuperação não devem derrubar a conversa: o Otis
		// ainda responde, só que sem o contexto extra da documentação.
		if text, err := s.retriever.Retrieve(ctx, message); err == nil {
			retrievedContext = text
		}
	}

	systemPrompt := BuildSystemPrompt(retrievedContext)

	turns := append(history, Message{Role: "user", Content: message})
	return s.ollama.Chat(ctx, systemPrompt, turns)
}
