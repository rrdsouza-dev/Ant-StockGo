package services

import (
	"errors"

	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

var ErrForbiddenDeposit = errors.New("você não tem acesso a este depósito")

// InventoryService concentra as regras de negócio de itens de estoque e
// suas movimentações. Toda operação passa primeiro por DepositService
// para confirmar que o usuário tem acesso ao depósito envolvido — é aqui
// que a regra "professor só acessa turmas vinculadas" vira, na prática,
// "professor só movimenta os depósitos das suas turmas".
type InventoryService struct {
	inventory *repositories.InventoryRepository
	deposits  *DepositService
}

func NewInventoryService(inventory *repositories.InventoryRepository, deposits *DepositService) *InventoryService {
	return &InventoryService{inventory: inventory, deposits: deposits}
}

// List retorna os itens de inventário do(s) depósito(s) que o usuário
// pode acessar. Se depositID for informado, restringe a esse depósito
// (após confirmar acesso); caso contrário, retorna todos os acessíveis.
func (s *InventoryService) List(user domain.User, depositID string) ([]domain.InventoryItem, error) {
	if depositID != "" {
		ok, err := s.deposits.CanAccess(user, depositID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrForbiddenDeposit
		}
		return s.inventory.ListByDeposits([]string{depositID})
	}

	deposits, err := s.deposits.List(user)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(deposits))
	for _, d := range deposits {
		ids = append(ids, d.ID)
	}
	return s.inventory.ListByDeposits(ids)
}

// Create cadastra um novo item de inventário em um depósito (somente
// gestão — aplicado pelo handler via middleware.RequireRole).
func (s *InventoryService) Create(user domain.User, depositID, name, sku string, minQuantity int) (domain.InventoryItem, error) {
	ok, err := s.deposits.CanAccess(user, depositID)
	if err != nil {
		return domain.InventoryItem{}, err
	}
	if !ok {
		return domain.InventoryItem{}, ErrForbiddenDeposit
	}
	if name == "" {
		return domain.InventoryItem{}, errors.New("nome do item é obrigatório")
	}
	return s.inventory.Create(domain.InventoryItem{DepositID: depositID, Name: name, SKU: sku, MinQuantity: minQuantity})
}

func (s *InventoryService) Update(user domain.User, itemID, name, sku string, minQuantity int) (domain.InventoryItem, error) {
	item, err := s.inventory.FindByID(itemID)
	if err != nil {
		return domain.InventoryItem{}, err
	}
	ok, err := s.deposits.CanAccess(user, item.DepositID)
	if err != nil {
		return domain.InventoryItem{}, err
	}
	if !ok {
		return domain.InventoryItem{}, ErrForbiddenDeposit
	}
	return s.inventory.Update(itemID, name, sku, minQuantity)
}

func (s *InventoryService) Deactivate(user domain.User, itemID string) error {
	item, err := s.inventory.FindByID(itemID)
	if err != nil {
		return err
	}
	ok, err := s.deposits.CanAccess(user, item.DepositID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbiddenDeposit
	}
	return s.inventory.Deactivate(itemID)
}

// MoveStock registra uma entrada ou saída de estoque. Disponível tanto
// para professor quanto gestão, desde que o depósito do item seja
// acessível ao usuário — é a operação central do perfil professor.
// Fluxo no sistema: chamado por POST /inventory/move. Gera sempre um
// StockMovement (auditoria) e atualiza a quantidade do item na mesma
// transação (ver InventoryRepository.Move).
func (s *InventoryService) MoveStock(user domain.User, itemID string, movementType domain.MovementType, quantity int, note string) (domain.InventoryItem, domain.StockMovement, error) {
	if quantity <= 0 {
		return domain.InventoryItem{}, domain.StockMovement{}, errors.New("quantidade deve ser maior que zero")
	}
	if !movementType.IsValid() {
		return domain.InventoryItem{}, domain.StockMovement{}, errors.New("tipo de movimentação inválido: use 'in' ou 'out'")
	}

	item, err := s.inventory.FindByID(itemID)
	if err != nil {
		return domain.InventoryItem{}, domain.StockMovement{}, err
	}

	ok, err := s.deposits.CanAccess(user, item.DepositID)
	if err != nil {
		return domain.InventoryItem{}, domain.StockMovement{}, err
	}
	if !ok {
		return domain.InventoryItem{}, domain.StockMovement{}, ErrForbiddenDeposit
	}

	return s.inventory.Move(item, movementType, quantity, note, user.ID)
}

// ListMovements retorna o histórico de movimentações visível ao usuário,
// opcionalmente restrito a um único depósito.
func (s *InventoryService) ListMovements(user domain.User, depositID string, limit int) ([]domain.StockMovement, error) {
	var ids []string
	if depositID != "" {
		ok, err := s.deposits.CanAccess(user, depositID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrForbiddenDeposit
		}
		ids = []string{depositID}
	} else {
		deposits, err := s.deposits.List(user)
		if err != nil {
			return nil, err
		}
		for _, d := range deposits {
			ids = append(ids, d.ID)
		}
	}
	return s.inventory.ListMovements(ids, limit)
}
