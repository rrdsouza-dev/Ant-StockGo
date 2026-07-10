package repositories

import (
	"database/sql"

	"github.com/lib/pq"
	"wms-backend/internal/domain"
)

const depositColumns = `id, name, description, is_administrative, active, created_by, created_at, updated_at`

// DepositRepository isola todo acesso SQL à tabela `deposits`.
type DepositRepository struct {
	db *sql.DB
}

func NewDepositRepository(db *sql.DB) *DepositRepository {
	return &DepositRepository{db: db}
}

// Create insere um novo depósito. Se isAdministrative for true, desmarca
// atomicamente qualquer outro depósito administrativo antes de inserir —
// só pode existir um por vez (ver migração 003 e o comentário em
// domain.Deposit.IsAdministrative).
func (r *DepositRepository) Create(name, description, createdBy string, isAdministrative bool) (domain.Deposit, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return domain.Deposit{}, err
	}
	defer tx.Rollback()

	if isAdministrative {
		if _, err := tx.Exec(`UPDATE deposits SET is_administrative = false WHERE is_administrative = true`); err != nil {
			return domain.Deposit{}, err
		}
	}

	row := tx.QueryRow(`
		INSERT INTO deposits (name, description, is_administrative, active, created_by)
		VALUES ($1, $2, $3, true, $4)
		RETURNING `+depositColumns,
		name, description, isAdministrative, createdBy,
	)
	created, err := scanDeposit(row)
	if err != nil {
		return domain.Deposit{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Deposit{}, err
	}
	return created, nil
}

// List retorna todos os depósitos ativos. Usado pela gestão para
// gerenciar (criar/editar/excluir) depósitos como entidade, e pelas
// turmas para escolher quais depósitos elas liberam — NÃO é usado para
// decidir acesso a estoque (ver ClassService.AccessibleDepositIDs).
func (r *DepositRepository) List() ([]domain.Deposit, error) {
	query := `SELECT ` + depositColumns + ` FROM deposits WHERE active = true ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Deposit
	for rows.Next() {
		d, err := scanDepositRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// ListByIDs retorna apenas os depósitos cujo id está no conjunto informado
// (usado para restringir o ESTOQUE acessível: professores às suas turmas,
// gestão ao depósito administrativo).
//
// O parâmetro precisa do cast explícito ::uuid[] — sem ele, o driver
// (lib/pq) envia o array como texto puro e o Postgres não resolve de
// forma confiável o tipo em `= ANY($1)` contra uma coluna uuid.
func (r *DepositRepository) ListByIDs(ids []string) ([]domain.Deposit, error) {
	if len(ids) == 0 {
		return []domain.Deposit{}, nil
	}
	query := `
		SELECT ` + depositColumns + `
		FROM deposits WHERE active = true AND id = ANY($1::uuid[]) ORDER BY created_at DESC`
	rows, err := r.db.Query(query, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Deposit
	for rows.Next() {
		d, err := scanDepositRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// AdministrativeIDs retorna o(s) id(s) de depósito(s) marcados como
// administrativos (na prática, no máximo um, garantido pela constraint
// única). É o conjunto de depósitos cujo estoque a gestão pode acessar.
func (r *DepositRepository) AdministrativeIDs() ([]string, error) {
	rows, err := r.db.Query(`SELECT id FROM deposits WHERE active = true AND is_administrative = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrings(rows)
}

func (r *DepositRepository) FindByID(id string) (domain.Deposit, error) {
	query := `SELECT ` + depositColumns + ` FROM deposits WHERE id = $1`
	row := r.db.QueryRow(query, id)
	return scanDeposit(row)
}

// Update altera nome/descrição/flag administrativa de um depósito. Se
// isAdministrative for true, desmarca atomicamente qualquer outro
// depósito que estivesse marcado antes de aplicar a mudança neste.
func (r *DepositRepository) Update(id, name, description string, isAdministrative bool) (domain.Deposit, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return domain.Deposit{}, err
	}
	defer tx.Rollback()

	if isAdministrative {
		if _, err := tx.Exec(`UPDATE deposits SET is_administrative = false WHERE is_administrative = true AND id != $1`, id); err != nil {
			return domain.Deposit{}, err
		}
	}

	row := tx.QueryRow(`
		UPDATE deposits SET name = $1, description = $2, is_administrative = $3, updated_at = now()
		WHERE id = $4
		RETURNING `+depositColumns,
		name, description, isAdministrative, id,
	)
	updated, err := scanDeposit(row)
	if err != nil {
		return domain.Deposit{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Deposit{}, err
	}
	return updated, nil
}

// Deactivate faz soft-delete: o depósito some das listagens mas o
// histórico de inventário e movimentações permanece intacto.
func (r *DepositRepository) Deactivate(id string) error {
	_, err := r.db.Exec(`UPDATE deposits SET active = false, updated_at = now() WHERE id = $1`, id)
	return err
}

func scanDeposit(row *sql.Row) (domain.Deposit, error) {
	var d domain.Deposit
	err := row.Scan(&d.ID, &d.Name, &d.Description, &d.IsAdministrative, &d.Active, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.Deposit{}, ErrNotFound
	}
	return d, err
}

func scanDepositRows(rows *sql.Rows) (domain.Deposit, error) {
	var d domain.Deposit
	err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.IsAdministrative, &d.Active, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}
