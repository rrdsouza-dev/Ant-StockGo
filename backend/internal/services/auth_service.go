package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"wms-backend/internal/auth"
	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
	"wms-backend/internal/validation"
)

var (
	ErrEmailInUse        = errors.New("este e-mail já está cadastrado ou aguardando aprovação")
	ErrInvalidCredentials = errors.New("e-mail ou senha inválidos")
	ErrAccountInactive    = errors.New("sua conta está desativada, contate a gestão")
	ErrInvalidRole        = errors.New("perfil inválido: use 'professor' ou 'gestao'")
)

// AuthService concentra as regras de negócio de autenticação e do fluxo
// de aprovação de contas (pendente -> aprovado/rejeitado -> ativo).
// Nenhum handler acessa repositories.UserRepository ou
// repositories.PendingUserRepository diretamente — sempre passa por aqui.
type AuthService struct {
	users    *repositories.UserRepository
	pending  *repositories.PendingUserRepository
	jwt      *auth.JWTManager
}

func NewAuthService(users *repositories.UserRepository, pending *repositories.PendingUserRepository, jwt *auth.JWTManager) *AuthService {
	return &AuthService{users: users, pending: pending, jwt: jwt}
}

// Register valida os dados de cadastro e cria uma solicitação PENDENTE.
// Fluxo no sistema: chamado por POST /auth/register. Nunca gera token e
// nunca ativa a conta — apenas a gestão pode fazer isso via ApproveOrReject.
// Efeitos colaterais: grava um novo registro em pending_users; a senha é
// hasheada com bcrypt antes de qualquer escrita.
func (s *AuthService) Register(name, email, password string, role domain.Role) (domain.PendingUser, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

	if !validation.Required(name) {
		return domain.PendingUser{}, errors.New("nome é obrigatório")
	}
	if !validation.IsValidEmail(email) {
		return domain.PendingUser{}, errors.New("e-mail inválido")
	}
	if ok, msg := validation.ValidatePassword(password); !ok {
		return domain.PendingUser{}, errors.New(msg)
	}
	if !role.IsValid() {
		return domain.PendingUser{}, ErrInvalidRole
	}

	if exists, err := s.users.EmailExists(email); err != nil {
		return domain.PendingUser{}, err
	} else if exists {
		return domain.PendingUser{}, ErrEmailInUse
	}
	if exists, err := s.pending.EmailPending(email); err != nil {
		return domain.PendingUser{}, err
	} else if exists {
		return domain.PendingUser{}, ErrEmailInUse
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return domain.PendingUser{}, fmt.Errorf("erro ao proteger a senha: %w", err)
	}

	return s.pending.Create(domain.PendingUser{
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	})
}

// Login valida as credenciais contra a tabela de usuários já aprovados e,
// se corretas, emite um JWT. Contas pendentes ou rejeitadas nunca aparecem
// em `users`, então falham aqui com ErrInvalidCredentials — o sistema não
// revela se um e-mail está pendente, ativo ou inexistente, por segurança.
func (s *AuthService) Login(email, password string) (domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.FindByEmail(email)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.User{}, "", ErrInvalidCredentials
		}
		return domain.User{}, "", err
	}

	if !auth.CheckPassword(user.PasswordHash, password) {
		return domain.User{}, "", ErrInvalidCredentials
	}
	if !user.Active {
		return domain.User{}, "", ErrAccountInactive
	}

	token, err := s.jwt.Generate(user)
	if err != nil {
		return domain.User{}, "", fmt.Errorf("erro ao gerar token: %w", err)
	}
	return user, token, nil
}

// ListPending retorna as solicitações aguardando decisão (painel da gestão).
// Fluxo no sistema: chamado por GET /auth/pending, protegido por
// middleware.RequireRole(gestao).
func (s *AuthService) ListPending() ([]domain.PendingUser, error) {
	return s.pending.ListPending()
}

// ApproveOrReject decide uma solicitação pendente.
// Se aprovada: cria o User real (ativado) e marca a solicitação como
// "approved". Se rejeitada: apenas marca como "rejected", sem criar conta.
// Fluxo no sistema: chamado por POST /auth/approve, protegido por
// middleware.RequireRole(gestao). É o único ponto do sistema que cria
// contas ativas a partir de um cadastro público.
func (s *AuthService) ApproveOrReject(pendingID string, approve bool) (*domain.User, error) {
	p, err := s.pending.FindByID(pendingID)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.PendingStatusPending {
		return nil, errors.New("esta solicitação já foi revisada")
	}

	if !approve {
		if err := s.pending.UpdateStatus(pendingID, domain.PendingStatusRejected); err != nil {
			return nil, err
		}
		return nil, nil
	}

	created, err := s.users.Create(domain.User{
		Name:         p.Name,
		Email:        p.Email,
		PasswordHash: p.PasswordHash,
		Role:         p.Role,
		SupportCode:  generateSupportCode(),
	})
	if err != nil {
		return nil, err
	}
	if err := s.pending.UpdateStatus(pendingID, domain.PendingStatusApproved); err != nil {
		return nil, err
	}
	return &created, nil
}

// generateSupportCode cria o código pessoal usado para validar a
// abertura de chamados de suporte (formato SKU-XXX-XXX). Gerado uma
// única vez, na aprovação da conta — visível ao próprio usuário via
// GET /users/me (campo support_code).
func generateSupportCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // sem O/0/I/1 para evitar ambiguidade visual
	segment := func() string {
		b := make([]byte, 3)
		buf := make([]byte, 3)
		_, _ = rand.Read(buf)
		for i, v := range buf {
			b[i] = alphabet[int(v)%len(alphabet)]
		}
		return string(b)
	}
	return fmt.Sprintf("SKU-%s-%s", segment(), segment())
}
