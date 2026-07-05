package domain

import "time"

// Role define o perfil de acesso do usuário no sistema.
type Role string

const (
	RoleProfessor Role = "professor"
	RoleGestao    Role = "gestao"
)

// IsValid confirma se o valor recebido corresponde a um perfil suportado.
func (r Role) IsValid() bool {
	return r == RoleProfessor || r == RoleGestao
}

// User representa uma conta ativa no sistema (já aprovada pela gestão).
//
// Fluxo no sistema: criado exclusivamente pelo AuthService.ApproveUser,
// nunca diretamente por cadastro público (ver PendingUser).
// Efeitos colaterais: nenhum ao ser apenas lido; escrita sempre passa
// pelo repository, que é a única camada que toca o banco.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PublicUser é a projeção segura do usuário exposta pela API (sem hash de senha).
// Inclui as turmas e depósitos vinculados quando o chamador solicitar (perfil "me").
type PublicUser struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	Classes   []Class   `json:"classes,omitempty"`
	Deposits  []Deposit `json:"deposits,omitempty"`
}

// ToPublic remove dados sensíveis antes de qualquer resposta HTTP.
func (u User) ToPublic() PublicUser {
	return PublicUser{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		Active:    u.Active,
		CreatedAt: u.CreatedAt,
	}
}
