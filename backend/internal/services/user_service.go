package services

import (
	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

// UserService monta a visão pública de um usuário, incluindo suas turmas
// e depósitos vinculados quando aplicável.
type UserService struct {
	users    *repositories.UserRepository
	classes  *ClassService
	deposits *DepositService
}

func NewUserService(users *repositories.UserRepository, classes *ClassService, deposits *DepositService) *UserService {
	return &UserService{users: users, classes: classes, deposits: deposits}
}

// Me monta o perfil completo do usuário autenticado: dados básicos +
// turmas vinculadas (professor) + depósitos acessíveis (via turmas para
// professor, todos os depósitos ativos para gestão).
// Fluxo no sistema: chamado por GET /users/me em toda sessão iniciada
// pelo frontend, para hidratar a sidebar e o seletor de depósito.
func (s *UserService) Me(user domain.User) (domain.PublicUser, error) {
	public := user.ToPublic()

	classes, err := s.classes.List(user)
	if err != nil {
		return domain.PublicUser{}, err
	}
	public.Classes = classes

	deposits, err := s.deposits.List(user)
	if err != nil {
		return domain.PublicUser{}, err
	}
	public.Deposits = deposits

	return public, nil
}

// List retorna todos os usuários ativos (somente gestão).
func (s *UserService) List() ([]domain.User, error) {
	return s.users.List()
}
