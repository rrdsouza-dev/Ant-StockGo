package services

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

var (
	ErrInvalidAdminCode = errors.New("código administrativo inválido")
)

const supportLockDuration = 15 * time.Second

// supportAttempt rastreia, em memória, as tentativas erradas do código de
// suporte de um usuário. É estado efêmero de propósito (não persistido):
// o objetivo é só desacelerar tentativas de chute dentro de uma mesma
// sessão do servidor, não é um mecanismo de segurança crítico — por isso
// um mapa protegido por mutex é suficiente, sem precisar de tabela nova.
type supportAttempt struct {
	count       int
	lockedUntil time.Time
}

// SupportService concentra as regras de negócio de chamados de suporte:
// validação do código pessoal do professor (com as mensagens e o
// bloqueio temporário exigidos pela especificação), listagem para a
// gestão e limpeza de histórico protegida por código administrativo.
type SupportService struct {
	tickets   *repositories.SupportTicketRepository
	adminCode string

	mu       sync.Mutex
	attempts map[string]*supportAttempt
}

func NewSupportService(tickets *repositories.SupportTicketRepository, adminCode string) *SupportService {
	return &SupportService{
		tickets:   tickets,
		adminCode: adminCode,
		attempts:  make(map[string]*supportAttempt),
	}
}

// TicketInput agrupa os dados enviados pelo professor ao abrir um chamado.
// Nome e e-mail NÃO vêm daqui — são sempre lidos do usuário autenticado
// (ver Create), conforme "nome e e-mail somente leitura (vindos do backend)".
type TicketInput struct {
	Categoria domain.SupportCategory
	Tipo      domain.SupportIssueType
	Codigo    string
	Descricao string
}

// Create valida o código de suporte do professor e, se correto, grava o
// chamado. Fluxo no sistema: chamado por POST /support/tickets, restrito
// a professor pelo middleware de rota.
//
// Validação do código: exclusivamente aqui no backend, nunca no frontend.
// 1ª tentativa errada -> "Tá errado mano."
// 2ª tentativa errada -> "JÁ FALEI QUE TA ERRADO MALUCO KKKK." + bloqueio
// temporário de novas tentativas.
func (s *SupportService) Create(user domain.User, input TicketInput) (domain.SupportTicket, error) {
	if !input.Categoria.IsValid() {
		return domain.SupportTicket{}, errors.New("categoria inválida: use 'estoque' ou 'sistema'")
	}
	if !input.Tipo.IsValid() {
		return domain.SupportTicket{}, errors.New("tipo inválido: use 'erro_operacional' ou 'erro_sistematico'")
	}

	if locked, waitSeconds := s.isLocked(user.ID); locked {
		return domain.SupportTicket{}, fmt.Errorf("aguarde %ds antes de tentar novamente", waitSeconds)
	}

	if !s.validCode(user, input.Codigo) {
		msg := s.registerFailedAttempt(user.ID)
		return domain.SupportTicket{}, errors.New(msg)
	}
	s.clearAttempts(user.ID)

	return s.tickets.Create(domain.SupportTicket{
		ProfessorID: user.ID,
		Nome:        user.Name,
		Email:       user.Email,
		Categoria:   input.Categoria,
		Tipo:        input.Tipo,
		Descricao:   strings.TrimSpace(input.Descricao),
	})
}

func (s *SupportService) validCode(user domain.User, codigo string) bool {
	codigo = strings.TrimSpace(codigo)
	return codigo != "" && user.SupportCode != "" && strings.EqualFold(codigo, user.SupportCode)
}

func (s *SupportService) isLocked(userID string) (bool, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.attempts[userID]
	if !ok || a.lockedUntil.IsZero() {
		return false, 0
	}
	remaining := time.Until(a.lockedUntil)
	if remaining <= 0 {
		return false, 0
	}
	return true, int(remaining.Seconds()) + 1
}

func (s *SupportService) registerFailedAttempt(userID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.attempts[userID]
	if !ok {
		a = &supportAttempt{}
		s.attempts[userID] = a
	}
	a.count++

	if a.count >= 2 {
		a.lockedUntil = time.Now().Add(supportLockDuration)
		return "JÁ FALEI QUE TA ERRADO MALUCO KKKK."
	}
	return "Tá errado mano."
}

func (s *SupportService) clearAttempts(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.attempts, userID)
}

// List retorna todos os chamados (somente gestão).
func (s *SupportService) List() ([]domain.SupportTicket, error) {
	return s.tickets.List()
}

// ClearHistory apaga todo o histórico de chamados, mas só depois de
// validar o código administrativo — nunca confiando em nada vindo do
// frontend além do próprio código a ser conferido.
func (s *SupportService) ClearHistory(adminCode string) error {
	if s.adminCode == "" || !strings.EqualFold(strings.TrimSpace(adminCode), s.adminCode) {
		return ErrInvalidAdminCode
	}
	return s.tickets.DeleteAll()
}
