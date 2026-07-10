package services

import (
	"errors"

	"wms-backend/internal/domain"
	"wms-backend/internal/repositories"
)

// ClassService concentra as regras de negócio de turmas, incluindo a
// regra central de escopo de acesso: "turmas definem acesso aos estoques".
type ClassService struct {
	classes  *repositories.ClassRepository
	deposits *repositories.DepositRepository
}

func NewClassService(classes *repositories.ClassRepository, deposits *repositories.DepositRepository) *ClassService {
	return &ClassService{classes: classes, deposits: deposits}
}

// AccessibleDepositIDs é a regra de autorização usada por TODOS os
// services que leem/escrevem ESTOQUE (itens de inventário e
// movimentações) — não confundir com a listagem de depósitos como
// entidade administrável (essa continua irrestrita para a gestão, ver
// DepositService.List).
//
// Regra atual (pós-especificação de permissões): a gestão só acessa o
// estoque do(s) depósito(s) marcado(s) como administrativo — ela NÃO
// enxerga o estoque das turmas, mesmo continuando responsável por
// criar/editar/excluir turmas e depósitos como entidades. O professor
// acessa os depósitos da turma "ativa" na sessão quando informada
// (classID), ou a união de todas as suas turmas quando não informada
// (ex.: telas que ainda não pedem uma turma específica).
//
// Fluxo no sistema: chamado por DepositService e InventoryService antes
// de qualquer leitura ou escrita, nunca pulado — é aqui que a regra
// "nenhuma regra de acesso pode existir no frontend" é aplicada de fato.
func (s *ClassService) AccessibleDepositIDs(user domain.User, classID string) ([]string, error) {
	if user.Role == domain.RoleGestao {
		return s.deposits.AdministrativeIDs()
	}
	if classID != "" {
		// A consulta já filtra por user_id E class_id juntos: se o
		// professor não estiver de fato vinculado à turma informada,
		// não retorna nada — nunca confia no classID vindo do cliente
		// sem checar o vínculo real no banco.
		return s.classes.DepositIDsForProfessorClass(user.ID, classID)
	}
	return s.classes.DepositIDsForProfessor(user.ID)
}

// Create cria uma nova turma (somente gestão, aplicado pelo handler/middleware).
func (s *ClassService) Create(name, description string, teacherIDs, depositIDs []string) (domain.Class, error) {
	if name == "" {
		return domain.Class{}, errors.New("nome da turma é obrigatório")
	}
	created, err := s.classes.Create(name, description)
	if err != nil {
		return domain.Class{}, err
	}
	if err := s.classes.SetTeachers(created.ID, teacherIDs); err != nil {
		return domain.Class{}, err
	}
	if err := s.classes.SetDeposits(created.ID, depositIDs); err != nil {
		return domain.Class{}, err
	}
	return s.hydrate(created)
}

// List retorna as turmas visíveis para o usuário: todas para a gestão,
// apenas as vinculadas para o professor.
func (s *ClassService) List(user domain.User) ([]domain.Class, error) {
	var list []domain.Class
	var err error
	if user.Role == domain.RoleGestao {
		list, err = s.classes.List()
	} else {
		list, err = s.classes.ListForProfessor(user.ID)
	}
	if err != nil {
		return nil, err
	}

	hydrated := make([]domain.Class, 0, len(list))
	for _, c := range list {
		h, err := s.hydrate(c)
		if err != nil {
			return nil, err
		}
		hydrated = append(hydrated, h)
	}
	return hydrated, nil
}

// Update atualiza dados da turma e, se informados, os vínculos de
// professores e depósitos (somente gestão).
func (s *ClassService) Update(id, name, description string, teacherIDs, depositIDs []string) (domain.Class, error) {
	updated, err := s.classes.Update(id, name, description)
	if err != nil {
		return domain.Class{}, err
	}
	if teacherIDs != nil {
		if err := s.classes.SetTeachers(id, teacherIDs); err != nil {
			return domain.Class{}, err
		}
	}
	if depositIDs != nil {
		if err := s.classes.SetDeposits(id, depositIDs); err != nil {
			return domain.Class{}, err
		}
	}
	return s.hydrate(updated)
}

func (s *ClassService) Delete(id string) error {
	return s.classes.Delete(id)
}

func (s *ClassService) hydrate(c domain.Class) (domain.Class, error) {
	teacherIDs, err := s.classes.TeacherIDs(c.ID)
	if err != nil {
		return domain.Class{}, err
	}
	depositIDs, err := s.classes.DepositIDs(c.ID)
	if err != nil {
		return domain.Class{}, err
	}
	deposits, err := s.deposits.ListByIDs(depositIDs)
	if err != nil {
		return domain.Class{}, err
	}
	c.TeacherIDs = teacherIDs
	c.Deposits = deposits
	return c, nil
}
