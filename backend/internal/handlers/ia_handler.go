package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/ia"
)

// OtisHandler expõe o chat do Otis. A rota é registrada com
// authRequired (ver routes.go) — nenhuma mensagem chega aqui sem um
// usuário autenticado válido, do mesmo jeito que qualquer outra rota
// da API.
type OtisHandler struct {
	otis *ia.OtisService
}

func NewOtisHandler(otis *ia.OtisService) *OtisHandler {
	return &OtisHandler{otis: otis}
}

type otisMessageInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type otisChatRequest struct {
	Message string             `json:"message"`
	History []otisMessageInput `json:"history"`
}

type otisChatResponse struct {
	Response string `json:"response"`
}

// Chat — POST /otis/chat
// Recebe a mensagem atual do usuário e, opcionalmente, o histórico da
// conversa (mantido apenas no frontend nesta V1 — ver documentação).
// Nunca lê identidade do corpo da requisição: quem está perguntando já
// foi resolvido pelo middleware de autenticação antes deste handler
// rodar, embora o Otis em si ainda não personalize a resposta por
// usuário nesta versão.
func (h *OtisHandler) Chat(c *gin.Context) {
	var req otisChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}

	history := make([]ia.Message, 0, len(req.History))
	for _, m := range req.History {
		history = append(history, ia.Message{Role: m.Role, Content: m.Content})
	}

	response, err := h.otis.Ask(c.Request.Context(), req.Message, history)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, ia.ErrEmptyMessage),
			errors.Is(err, ia.ErrMessageTooLong),
			errors.Is(err, ia.ErrTooManyHistoryMessages):
			status = http.StatusBadRequest
		default:
			// Falha de comunicação com o Ollama ou resposta inesperada.
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": errorMessageFor(err, status)})
		return
	}

	c.JSON(http.StatusOK, otisChatResponse{Response: response})
}

// errorMessageFor evita vazar detalhes internos (URL do Ollama, stack de
// erro de rede) para o frontend quando a falha não é de validação —
// nesse caso devolve uma mensagem genérica e estável.
func errorMessageFor(err error, status int) string {
	if status == http.StatusBadRequest {
		return err.Error()
	}
	return "Otis está indisponível no momento. Tente novamente em instantes."
}
