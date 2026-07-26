package services

import (
	"errors"
	"strings"

	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

var ErrBarcodeInUse = errors.New("este código de barras já está associado a outro Pré-Produto")

// PreProductInput agrupa os campos graváveis de um Pré-Produto.
type PreProductInput struct {
	Name       string
	CategoryID *string
	Brand      string
	Unit       string
	Notes      string
}

// PreProductService concentra as regras de negócio do catálogo de
// Pré-Produtos: cadastro do "molde" reutilizável (sem quantidade,
// validade, lote ou código de barras fixo — esses dados pertencem
// exclusivamente ao item de estoque real, criado a partir dele) e a
// associação opcional de um código de barras para reconhecimento
// automático no scanner.
type PreProductService struct {
	preProducts *repositories.PreProductRepository
}

func NewPreProductService(preProducts *repositories.PreProductRepository) *PreProductService {
	return &PreProductService{preProducts: preProducts}
}

// Create cadastra um novo Pré-Produto. Disponível para qualquer usuário
// autenticado (professor e gestão), assim como a criação de itens de
// estoque e categorias.
func (s *PreProductService) Create(userID string, input PreProductInput) (domain.PreProduct, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.PreProduct{}, errors.New("nome é obrigatório")
	}
	unit := strings.TrimSpace(input.Unit)
	if unit == "" {
		return domain.PreProduct{}, errors.New("unidade de medida é obrigatória")
	}

	return s.preProducts.Create(domain.PreProduct{
		Name:       name,
		CategoryID: input.CategoryID,
		Brand:      strings.TrimSpace(input.Brand),
		Unit:       unit,
		Notes:      strings.TrimSpace(input.Notes),
		CreatedBy:  userID,
	})
}

// List retorna todo o catálogo de Pré-Produtos ativos.
func (s *PreProductService) List() ([]domain.PreProduct, error) {
	return s.preProducts.List()
}

// FindByBarcode localiza o Pré-Produto associado a um código de barras.
// Usado pelo fluxo do scanner: antes de perguntar ao usuário a qual
// Pré-Produto associar um código novo, o sistema confere se aquele
// código já foi associado anteriormente.
func (s *PreProductService) FindByBarcode(barcode string) (domain.PreProduct, error) {
	return s.preProducts.FindByBarcode(barcode)
}

func (s *PreProductService) Update(id string, input PreProductInput) (domain.PreProduct, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.PreProduct{}, errors.New("nome é obrigatório")
	}
	unit := strings.TrimSpace(input.Unit)
	if unit == "" {
		return domain.PreProduct{}, errors.New("unidade de medida é obrigatória")
	}
	return s.preProducts.Update(id, name, input.CategoryID, strings.TrimSpace(input.Brand), unit, strings.TrimSpace(input.Notes))
}

// AssociateBarcode vincula um código de barras a um Pré-Produto já
// existente — é o passo final do fluxo "código não encontrado" do
// scanner (ver item 6 da especificação). Garante que o código não esteja
// em uso por outro Pré-Produto antes de gravar.
func (s *PreProductService) AssociateBarcode(id, barcode string) (domain.PreProduct, error) {
	barcode = strings.TrimSpace(barcode)
	if barcode == "" {
		return domain.PreProduct{}, errors.New("código de barras é obrigatório")
	}
	existing, err := s.preProducts.FindByBarcode(barcode)
	if err == nil && existing.ID != id {
		return domain.PreProduct{}, ErrBarcodeInUse
	}
	if err != nil && err != repositories.ErrNotFound {
		return domain.PreProduct{}, err
	}
	return s.preProducts.SetBarcode(id, barcode)
}

func (s *PreProductService) Deactivate(id string) error {
	return s.preProducts.Deactivate(id)
}
