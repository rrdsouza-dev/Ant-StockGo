package repositories

import (
	"database/sql"

	"github.com/lib/pq"
	"wms-backend/internal/domain"
)

// DepositRepository isola todo acesso SQL à tabela `deposits`.
type DepositRepository struct {
	db *sql.DB
}

func NewDepositRepository(db *sql.DB) *DepositRepository {
	return &DepositRepository{db: db}
}

func (r *DepositRepository) Create(name, description, createdBy string) (domain.Deposit, error) {
	query := `
		INSERT INTO deposits (name, description, active, created_by)
		VALUES ($1, $2, true, $3)
		RETURNING id, name, description, active, created_by, created_at, updated_at`
	row := r.db.QueryRow(query, name, description, createdBy)
	return scanDeposit(row)
}

// List retorna todos os depósitos ativos (usado pela gestão, que vê tudo).
func (r *DepositRepository) List() ([]domain.Deposit, error) {
	query := `
		SELECT id, name, description, active, created_by, created_at, updated_at
		FROM deposits WHERE active = true ORDER BY created_at DESC`
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
// (usado para restringir professores aos depósitos das suas turmas).
//
// NOTA DE CORREÇÃO: o parâmetro precisa do cast explícito ::uuid[]. Sem
// ele, o driver (lib/pq) envia o array como texto puro e o Postgres não
// consegue resolver de forma confiável o tipo do parâmetro em `= ANY($1)`
// contra uma coluna uuid — o resultado, dependendo do plano gerado, é
// zero linhas ao invés de um erro. Era exatamente isso que fazia o
// professor "não ver" depósitos que estavam corretamente vinculados no
// banco: o filtro de acesso pegava os IDs certos (DepositIDsForProfessor),
// mas esta consulta seguinte não conseguia casar esses IDs com a tabela.
// A gestão nunca foi afetada porque DepositRepository.List() não usa
// parâmetro nenhum (lista tudo diretamente).
func (r *DepositRepository) ListByIDs(ids []string) ([]domain.Deposit, error) {
	if len(ids) == 0 {
		return []domain.Deposit{}, nil
	}
	query := `
		SELECT id, name, description, active, created_by, created_at, updated_at
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

func (r *DepositRepository) FindByID(id string) (domain.Deposit, error) {
	query := `
		SELECT id, name, description, active, created_by, created_at, updated_at
		FROM deposits WHERE id = $1`
	row := r.db.QueryRow(query, id)
	return scanDeposit(row)
}

func (r *DepositRepository) Update(id, name, description string) (domain.Deposit, error) {
	query := `
		UPDATE deposits SET name = $1, description = $2, updated_at = now()
		WHERE id = $3
		RETURNING id, name, description, active, created_by, created_at, updated_at`
	row := r.db.QueryRow(query, name, description, id)
	return scanDeposit(row)
}

// Deactivate faz soft-delete: o depósito some das listagens mas o
// histórico de inventário e movimentações permanece intacto.
func (r *DepositRepository) Deactivate(id string) error {
	_, err := r.db.Exec(`UPDATE deposits SET active = false, updated_at = now() WHERE id = $1`, id)
	return err
}

func scanDeposit(row *sql.Row) (domain.Deposit, error) {
	var d domain.Deposit
	err := row.Scan(&d.ID, &d.Name, &d.Description, &d.Active, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.Deposit{}, ErrNotFound
	}
	return d, err
}

func scanDepositRows(rows *sql.Rows) (domain.Deposit, error) {
	var d domain.Deposit
	err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.Active, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}
