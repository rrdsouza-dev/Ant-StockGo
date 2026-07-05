package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword aplica bcrypt (custo padrão) sobre a senha em texto puro.
// Fluxo no sistema: chamado exclusivamente pelo AuthService antes de
// persistir qualquer senha (cadastro). O texto puro NUNCA é armazenado.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compara a senha em texto puro informada no login com o
// hash armazenado. Retorna false para qualquer senha incorreta, sem
// vazar detalhes do motivo da falha.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
