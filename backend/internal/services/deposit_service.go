package services

import (
	"errors"

	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

// DepositService aplica as regras de negócio sobre depósitos.
//
// Há duas visões deliberadamente distintas neste service, que não devem
// ser confundidas:
//   - List (entidade administrável): quem pode criar/editar/excluir um
//     depósito como registro. A gestão continua vendo e gerenciando TODOS
//     os depósitos aqui — é o que alimenta a tela "Depósitos" e o seletor
//     de depósitos ao vincular uma turma.
//   - ListForStock / CanAccess (acesso ao ESTOQUE): quais depósitos um
//     usuário pode ler/mexer itens e movimentações. A gestão só acessa o
//     depósito administrativo; o professor só acessa os depósitos das
//     suas turmas (opcionalmente restrito a uma única turma "ativa").
type DepositService struct {
	deposits *repositories.DepositRepository
	classes  *ClassService
}

func NewDepositService(deposits *repositories.DepositRepository, classes *ClassService) *DepositService {
	return &DepositService{deposits: deposits, classes: classes}
}

// Create cadastra um novo depósito. Chamado apenas por handlers já
// protegidos por middleware.RequireRole(gestao). isAdministrative marca
// (de forma exclusiva — ver DepositRepository.Create) o depósito de uso
// da gestão.
func (s *DepositService) Create(name, description, createdBy string, isAdministrative bool) (domain.Deposit, error) {
	if name == "" {
		return domain.Deposit{}, errors.New("nome do depósito é obrigatório")
	}
	return s.deposits.Create(name, description, createdBy, isAdministrative)
}

// List retorna os depósitos como ENTIDADE administrável: todos para a
// gestão (que continua gerenciando qualquer depósito), apenas os
// vinculados às turmas para o professor.
func (s *DepositService) List(user domain.User) ([]domain.Deposit, error) {
	if user.Role == domain.RoleGestao {
		return s.deposits.List()
	}
	ids, err := s.classes.AccessibleDepositIDs(user, "")
	if err != nil {
		return nil, err
	}
	return s.deposits.ListByIDs(ids)
}

// ListForStock retorna os depósitos cujo ESTOQUE o usuário pode acessar:
// para a gestão, apenas o depósito administrativo; para o professor, os
// depósitos da turma "ativa" (classID) ou a união de todas as suas
// turmas quando classID vier vazio. Usada pelas telas de Estoque,
// Movimentações, Relatórios, Exportações e Dashboard.
func (s *DepositService) ListForStock(user domain.User, classID string) ([]domain.Deposit, error) {
	ids, err := s.classes.AccessibleDepositIDs(user, classID)
	if err != nil {
		return nil, err
	}
	return s.deposits.ListByIDs(ids)
}

// CanAccess confirma se o usuário pode ler/movimentar o ESTOQUE de um
// depósito específico — usado pelo InventoryService antes de qualquer
// leitura ou escrita de item/movimentação. Deliberadamente NÃO dá
// passe-livre para gestão: ela só acessa o depósito administrativo,
// exatamente como o professor só acessa os das suas turmas.
func (s *DepositService) CanAccess(user domain.User, depositID, classID string) (bool, error) {
	ids, err := s.classes.AccessibleDepositIDs(user, classID)
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

func (s *DepositService) Update(id, name, description string, isAdministrative bool) (domain.Deposit, error) {
	return s.deposits.Update(id, name, description, isAdministrative)
}

func (s *DepositService) Deactivate(id string) error {
	return s.deposits.Deactivate(id)
}
