package validation

import (
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)

// IsValidEmail confirma formato de e-mail. A duplicidade (já cadastrado)
// é verificada separadamente pelo repository, que consulta o banco.
func IsValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	return email != "" && emailRegex.MatchString(email)
}

// ValidatePassword aplica as regras obrigatórias de senha do sistema:
// mínimo 8 caracteres, 1 maiúscula, 1 número, 1 símbolo.
// Fluxo no sistema: chamado pelo AuthService no cadastro. É a validação
// definitiva — o frontend só replica isto para UX, nunca é confiável.
func ValidatePassword(password string) (bool, string) {
	if len(password) < 8 {
		return false, "a senha deve ter ao menos 8 caracteres"
	}
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`\d`).MatchString(password)
	hasSymbol := regexp.MustCompile(`[^A-Za-z0-9]`).MatchString(password)

	if !hasUpper {
		return false, "a senha deve conter ao menos 1 letra maiúscula"
	}
	if !hasDigit {
		return false, "a senha deve conter ao menos 1 número"
	}
	if !hasSymbol {
		return false, "a senha deve conter ao menos 1 símbolo"
	}
	return true, ""
}

// Required confirma que um campo de texto obrigatório não está vazio.
func Required(value string) bool {
	return strings.TrimSpace(value) != ""
}
