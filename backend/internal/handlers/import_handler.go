package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"wms-backend/internal/importing"
	"wms-backend/internal/middleware"
)

// ImportHandler expõe a Importação Inteligente: POST /imports/preview
// (recebe a planilha, devolve a prévia interpretada, nada é gravado) e
// POST /imports/commit (recebe os itens já revisados pelo usuário,
// grava via os Services existentes). Disponível para qualquer usuário
// autenticado com acesso de escrita ao depósito informado — a
// autorização real acontece dentro de InventoryService/PreProductService,
// nunca aqui (mesmo padrão do InventoryHandler).
type ImportHandler struct {
	imports *importing.ImportService
}

func NewImportHandler(imports *importing.ImportService) *ImportHandler {
	return &ImportHandler{imports: imports}
}

// acceptedExtensions é usado apenas para a mensagem de erro amigável
// quando a extensão do arquivo já denuncia um formato não suportado
// antes mesmo de olhar o conteúdo — a validação real de formato é
// sempre por conteúdo (ver importing.ParseSpreadsheet), nunca só pela
// extensão (item 4/25 da especificação).
var acceptedExtensions = []string{".xlsx", ".csv"}

// Preview — POST /imports/preview (multipart/form-data)
// Campos do form: "file" (obrigatório), "target" ("inventory_item" ou
// "pre_product", padrão "inventory_item" — ver item 13 da especificação).
func (h *ImportHandler) Preview(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "envie um arquivo no campo 'file'"})
		return
	}

	if fileHeader.Size > importing.MaxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": importing.ErrFileTooLarge.Error()})
		return
	}

	target := parseTargetKind(c.PostForm("target"))

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "não foi possível abrir o arquivo enviado"})
		return
	}
	defer file.Close()

	data, err := importing.ReadUploadLimited(file)
	if err != nil {
		respondImportError(c, err)
		return
	}

	// Timeout dedicado para a chamada de IA (resolução de cabeçalhos
	// ambíguos) não deixar a requisição de preview pendurada
	// indefinidamente se o Ollama estiver lento/indisponível — a
	// Importação Inteligente deve continuar funcionando mesmo assim
	// (item 20 da especificação), só sem a ajuda da IA para os
	// cabeçalhos que ficarem sem resolver.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	preview, err := h.imports.Preview(ctx, fileHeader.Filename, data, target)
	if err != nil {
		respondImportError(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}

type commitItemRequest struct {
	RowIndex    int     `json:"row_index"`
	Name        string  `json:"name"`
	Quantity    int     `json:"quantity"`
	LotNumber   string  `json:"lot_number"`
	ExpiryDate  string  `json:"expiry_date"`
	SKU         string  `json:"sku"`
	Brand       string  `json:"brand"`
	MinQuantity int     `json:"min_quantity"`
	CategoryID  *string `json:"category_id"`
	Notes       string  `json:"notes"`
	Unit        string  `json:"unit"`
}

type commitRequest struct {
	DepositID string              `json:"deposit_id"`
	Target    string              `json:"target"`
	Items     []commitItemRequest `json:"items"`
}

// maxCommitItems limita quantos itens podem ser gravados em uma única
// requisição de commit — protege contra um corpo de requisição forjado
// desproporcional ao que qualquer planilha real (já limitada a
// importing.MaxDataRows na etapa de preview) produziria (item 25 da
// especificação: nunca confiar nos dados enviados pela planilha, e por
// extensão, nunca confiar cegamente no corpo enviado ao commit).
const maxCommitItems = importing.MaxDataRows

// Commit — POST /imports/commit
// Grava, um a um, os itens já revisados pelo usuário na tela de prévia.
// Cada item é uma operação independente: uma falha em um item nunca
// interrompe os demais (ver ImportService.Commit).
func (h *ImportHandler) Commit(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sessão inválida"})
		return
	}

	var req commitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}

	if strings.TrimSpace(req.DepositID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "selecione um depósito para importar"})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nenhum item para importar"})
		return
	}
	if len(req.Items) > maxCommitItems {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantidade de itens excede o máximo permitido por importação"})
		return
	}

	target := parseTargetKind(req.Target)

	items := make([]importing.CommitItemInput, len(req.Items))
	for i, it := range req.Items {
		items[i] = importing.CommitItemInput{
			RowIndex:    it.RowIndex,
			Name:        it.Name,
			Quantity:    it.Quantity,
			LotNumber:   it.LotNumber,
			ExpiryDate:  it.ExpiryDate,
			SKU:         it.SKU,
			Brand:       it.Brand,
			MinQuantity: it.MinQuantity,
			CategoryID:  it.CategoryID,
			Notes:       it.Notes,
			Unit:        it.Unit,
		}
	}

	// O commit em si não depende de IA — sem necessidade de timeout
	// dedicado além do padrão da requisição HTTP; cada item é gravado
	// via chamadas diretas aos Services existentes (Postgres).
	result := h.imports.Commit(c.Request.Context(), user, importing.CommitRequest{
		DepositID: req.DepositID,
		Target:    target,
		Items:     items,
	})
	c.JSON(http.StatusOK, result)
}

func parseTargetKind(raw string) importing.TargetKind {
	if importing.TargetKind(raw) == importing.TargetPreProduct {
		return importing.TargetPreProduct
	}
	return importing.TargetInventoryItem // padrão: item 13 da especificação
}

// respondImportError traduz os erros conhecidos do pacote importing
// (parser/validação de upload) em códigos HTTP apropriados, seguindo o
// mesmo padrão de respondServiceError já usado pelos demais handlers.
func respondImportError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, importing.ErrFileTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
	case errors.Is(err, importing.ErrEmptyFile),
		errors.Is(err, importing.ErrNoDataRows),
		errors.Is(err, importing.ErrCorruptFile),
		errors.Is(err, importing.ErrUnsupportedFormat):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}
