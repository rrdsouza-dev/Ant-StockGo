package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient encapsula toda comunicação HTTP com o Ollama local.
// É a ÚNICA camada do sistema autorizada a falar com o Ollama — o
// frontend nunca acessa 127.0.0.1:11434 diretamente, e nenhuma outra
// parte do backend deve montar essa requisição.
//
// Fluxo no sistema: instanciado uma vez em main.go a partir de
// config.Config e injetado no OtisService, do mesmo jeito que um
// repository é injetado em um service.
type OllamaClient struct {
	baseURL string
	model   string
	http    *http.Client
}

// NewOllamaClient cria o cliente a partir de configuração explícita —
// nunca lê variáveis de ambiente diretamente (isso é responsabilidade
// exclusiva de config.Load()).
func NewOllamaClient(baseURL, model string, timeout time.Duration) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{Timeout: timeout},
	}
}

// chatMessage espelha o formato de mensagem da API /api/chat do Ollama.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponseMessage struct {
	Content string `json:"content"`
}

type chatResponse struct {
	Message chatResponseMessage `json:"message"`
	Done    bool                `json:"done"`
}

// Chat envia o histórico de mensagens (já incluindo o system prompt, ver
// prompt.go) ao Ollama e retorna o texto de resposta do modelo.
//
// Fluxo no sistema: chamado exclusivamente por OtisService.Ask.
// Efeito colateral: uma requisição HTTP de saída para OLLAMA_BASE_URL;
// nenhum dado é persistido aqui.
func (c *OllamaClient) Chat(ctx context.Context, systemPrompt string, history []Message) (string, error) {
	messages := make([]chatMessage, 0, len(history)+1)
	messages = append(messages, chatMessage{Role: "system", Content: systemPrompt})
	for _, m := range history {
		messages = append(messages, chatMessage{Role: m.Role, Content: m.Content})
	}

	payload := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("erro ao montar requisição para o Ollama: %w", err)
	}

	url := c.baseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisição para o Ollama: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("não foi possível conectar ao Ollama: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("erro ao ler resposta do Ollama: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama retornou status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("erro ao interpretar resposta do Ollama: %w", err)
	}

	if parsed.Message.Content == "" {
		return "", fmt.Errorf("ollama retornou uma resposta vazia")
	}

	return parsed.Message.Content, nil
}
