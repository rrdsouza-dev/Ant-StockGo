package services

import (
	"errors"
	"strings"

	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

var ErrCategoryInUse = errors.New("já existe uma categoria com este nome")
var ErrCategoryHasLinkedItems = errors.New("esta categoria está sendo utilizada por itens de estoque ou Pré-Produtos e não pode ser excluída")

// CategoryService concentra as regras de negócio de categorias de item
// de estoque. É uma entidade simples de propósito: existir para o
// formulário de item poder classificar itens e permitir que qualquer
// usuário autenticado (gestão ou professor) cadastre novas categorias
// sem precisar de deploy.
type CategoryService struct {
	categories *repositories.CategoryRepository
}

func NewCategoryService(categories *repositories.CategoryRepository) *CategoryService {
	return &CategoryService{categories: categories}
}

// Create cadastra uma nova categoria (chamado pelo botão "+" no formulário
// de item). Rejeita nomes vazios ou duplicados antes de tocar o banco.
func (s *CategoryService) Create(name string) (domain.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Category{}, errors.New("nome da categoria é obrigatório")
	}
	exists, err := s.categories.NameExists(name)
	if err != nil {
		return domain.Category{}, err
	}
	if exists {
		return domain.Category{}, ErrCategoryInUse
	}
	return s.categories.Create(name)
}

// List retorna todas as categorias cadastradas, para popular o <select>
// do formulário de item de estoque.
func (s *CategoryService) List() ([]domain.Category, error) {
	return s.categories.List()
}

// Delete remove uma categoria (somente gestão — restrição aplicada pelo
// middleware de rota). Antes de excluir, verifica se algum item de
// estoque ou Pré-Produto ainda referencia esta categoria — se sim, a
// exclusão é recusada com uma mensagem clara (ErrCategoryHasLinkedItems)
// em vez de deixar esses registros ficarem sem categoria silenciosamente.
func (s *CategoryService) Delete(id string) error {
	count, err := s.categories.CountUsage(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrCategoryHasLinkedItems
	}
	return s.categories.Delete(id)
}
