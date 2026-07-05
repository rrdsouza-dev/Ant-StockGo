package validation

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
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

var dateFieldRegex = regexp.MustCompile(`^(\d{2})/(\d{2})/(\d{4})$`)

// ParseBRDate valida e converte uma data no formato DD/MM/AAAA.
//
// Feito manualmente (sem depender de time.Parse) para ter controle total
// sobre a rejeição de dias fora do intervalo do mês (ex.: 31/02) — regra
// explícita da especificação de "data de validade" dos itens de estoque.
// Fluxo no sistema: chamado por InventoryService.Create/Update antes de
// persistir; é a validação que realmente vale — o frontend só replica a
// mesma regra para dar feedback imediato (com uma mensagem mais leve).
func ParseBRDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	m := dateFieldRegex.FindStringSubmatch(value)
	if m == nil {
		return time.Time{}, errors.New("data deve estar no formato DD/MM/AAAA")
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	year, _ := strconv.Atoi(m[3])

	if month < 1 || month > 12 {
		return time.Time{}, errors.New("mês inválido: deve estar entre 01 e 12")
	}
	if day < 1 || day > 31 {
		return time.Time{}, errors.New("dia inválido: deve estar entre 01 e 31")
	}

	daysInMonth := [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	max := daysInMonth[month-1]
	if month == 2 && isLeapYear(year) {
		max = 29
	}
	if day > max {
		return time.Time{}, errors.New("essa data não existe")
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

func isLeapYear(year int) bool {
	return (year%4 == 0 && year%100 != 0) || year%400 == 0
}
