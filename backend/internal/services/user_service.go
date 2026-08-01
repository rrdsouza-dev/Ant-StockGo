package services

import (
	"errors"

	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

var ErrCannotDeleteNonProfessor = errors.New("esta funcionalidade só permite excluir contas de professor")

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

// DeleteProfessor exclui permanentemente uma conta de professor.
// Deliberadamente restrito a contas com papel "professor" — mesmo que a
// rota já seja protegida para gestão, esta checagem no service impede
// que o próprio endpoint seja usado para excluir uma conta de gestão
// (inclusive por engano, passando o id errado). Registros que o
// professor tenha criado (movimentações, depósitos, Pré-Produtos,
// chamados de suporte) são preservados — ver migração 005.
func (s *UserService) DeleteProfessor(id string) error {
	target, err := s.users.FindByID(id)
	if err != nil {
		return err
	}
	if target.Role != domain.RoleProfessor {
		return ErrCannotDeleteNonProfessor
	}
	return s.users.Delete(id)
}
