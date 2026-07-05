package services

import (
	"errors"

	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

// DepositService aplica as regras de negócio sobre depósitos: apenas a
// gestão pode criar/editar/desativar; professores só podem visualizar os
// depósitos vinculados às suas turmas (via ClassService.AccessibleDepositIDs).
type DepositService struct {
	deposits *repositories.DepositRepository
	classes  *ClassService
}

func NewDepositService(deposits *repositories.DepositRepository, classes *ClassService) *DepositService {
	return &DepositService{deposits: deposits, classes: classes}
}

// Create cadastra um novo depósito. Chamado apenas por handlers já
// protegidos por middleware.RequireRole(gestao).
func (s *DepositService) Create(name, description, createdBy string) (domain.Deposit, error) {
	if name == "" {
		return domain.Deposit{}, errors.New("nome do depósito é obrigatório")
	}
	return s.deposits.Create(name, description, createdBy)
}

// List retorna os depósitos visíveis para o usuário autenticado, já
// filtrados pela regra de turmas quando o perfil é professor.
func (s *DepositService) List(user domain.User) ([]domain.Deposit, error) {
	ids, err := s.classes.AccessibleDepositIDs(user)
	if err != nil {
		return nil, err
	}
	if user.Role == domain.RoleGestao {
		return s.deposits.List()
	}
	return s.deposits.ListByIDs(ids)
}

// CanAccess confirma se o usuário pode ler/movimentar um depósito
// específico — usado pelo InventoryService antes de qualquer movimentação.
func (s *DepositService) CanAccess(user domain.User, depositID string) (bool, error) {
	if user.Role == domain.RoleGestao {
		return true, nil
	}
	ids, err := s.classes.AccessibleDepositIDs(user)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == depositID {
			return true, nil
		}
	}
	return false, nil
}

func (s *DepositService) Update(id, name, description string) (domain.Deposit, error) {
	return s.deposits.Update(id, name, description)
}

func (s *DepositService) Deactivate(id string) error {
	return s.deposits.Deactivate(id)
}
