package repositories

import (
	"database/sql"

	"wms-backend/internal/domain"
)

// ClassRepository isola todo acesso SQL à tabela `classes` e às tabelas
// de junção `class_teachers` (professores vinculados) e `class_deposits`
// (depósitos aos quais a turma dá acesso).
type ClassRepository struct {
	db *sql.DB
}

func NewClassRepository(db *sql.DB) *ClassRepository {
	return &ClassRepository{db: db}
}

func (r *ClassRepository) Create(name, description string) (domain.Class, error) {
	query := `
		INSERT INTO classes (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, created_at, updated_at`
	row := r.db.QueryRow(query, name, description)
	return scanClass(row)
}

func (r *ClassRepository) FindByID(id string) (domain.Class, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM classes WHERE id = $1`
	row := r.db.QueryRow(query, id)
	return scanClass(row)
}

// List retorna todas as turmas (visão da gestão).
func (r *ClassRepository) List() ([]domain.Class, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM classes ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Class
	for rows.Next() {
		c, err := scanClassRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// ListForProfessor retorna apenas as turmas às quais o professor pertence.
func (r *ClassRepository) ListForProfessor(userID string) ([]domain.Class, error) {
	query := `
		SELECT c.id, c.name, c.description, c.created_at, c.updated_at
		FROM classes c
		JOIN class_teachers ct ON ct.class_id = c.id
		WHERE ct.user_id = $1
		ORDER BY c.created_at DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Class
	for rows.Next() {
		c, err := scanClassRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *ClassRepository) Update(id, name, description string) (domain.Class, error) {
	query := `
		UPDATE classes SET name = $1, description = $2, updated_at = now()
		WHERE id = $3
		RETURNING id, name, description, created_at, updated_at`
	row := r.db.QueryRow(query, name, description, id)
	return scanClass(row)
}

func (r *ClassRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM classes WHERE id = $1`, id)
	return err
}

// SetTeachers substitui, de forma atômica, a lista de professores vinculados
// à turma (usado pelo formulário de edição de turma na gestão).
func (r *ClassRepository) SetTeachers(classID string, teacherIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM class_teachers WHERE class_id = $1`, classID); err != nil {
		return err
	}
	for _, teacherID := range teacherIDs {
		if _, err := tx.Exec(
			`INSERT INTO class_teachers (class_id, user_id) VALUES ($1, $2)`, classID, teacherID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetDeposits substitui, de forma atômica, os depósitos aos quais a turma
// dá acesso.
func (r *ClassRepository) SetDeposits(classID string, depositIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM class_deposits WHERE class_id = $1`, classID); err != nil {
		return err
	}
	for _, depositID := range depositIDs {
		if _, err := tx.Exec(
			`INSERT INTO class_deposits (class_id, deposit_id) VALUES ($1, $2)`, classID, depositID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// TeacherIDs retorna os professores vinculados a uma turma.
func (r *ClassRepository) TeacherIDs(classID string) ([]string, error) {
	rows, err := r.db.Query(`SELECT user_id FROM class_teachers WHERE class_id = $1`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrings(rows)
}

// DepositIDs retorna os depósitos vinculados a uma turma.
func (r *ClassRepository) DepositIDs(classID string) ([]string, error) {
	rows, err := r.db.Query(`SELECT deposit_id FROM class_deposits WHERE class_id = $1`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrings(rows)
}

// DepositIDsForProfessor retorna, sem duplicatas, todos os depósitos
// alcançáveis pelo professor através de QUALQUER turma à qual pertence.
// Esta é a regra de negócio "turmas definem acesso aos estoques" —
// usada como união quando nenhuma turma específica está "ativa" na sessão.
func (r *ClassRepository) DepositIDsForProfessor(userID string) ([]string, error) {
	query := `
		SELECT DISTINCT cd.deposit_id
		FROM class_deposits cd
		JOIN class_teachers ct ON ct.class_id = cd.class_id
		WHERE ct.user_id = $1`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrings(rows)
}

// DepositIDsForProfessorClass retorna os depósitos de UMA turma
// específica, mas só se o professor de fato estiver vinculado a ela —
// o JOIN com class_teachers filtrando por user_id é o que garante isso;
// se o professor não pertence à turma informada, a consulta não retorna
// nada (nunca confia no class_id vindo do cliente sem checar o vínculo).
// Usado quando o professor "escolhe uma turma" para trabalhar na sessão
// (ver item "Escolha de turma" da especificação).
func (r *ClassRepository) DepositIDsForProfessorClass(userID, classID string) ([]string, error) {
	query := `
		SELECT DISTINCT cd.deposit_id
		FROM class_deposits cd
		JOIN class_teachers ct ON ct.class_id = cd.class_id
		WHERE ct.user_id = $1 AND cd.class_id = $2`
	rows, err := r.db.Query(query, userID, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrings(rows)
}

func scanStrings(rows *sql.Rows) ([]string, error) {
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanClass(row *sql.Row) (domain.Class, error) {
	var c domain.Class
	err := row.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.Class{}, ErrNotFound
	}
	return c, err
}

func scanClassRows(rows *sql.Rows) (domain.Class, error) {
	var c domain.Class
	err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}
