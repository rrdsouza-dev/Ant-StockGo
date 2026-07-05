package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"wms-backend/internal/domain"
)

var ErrInvalidToken = errors.New("token inválido ou expirado")

// Claims são os dados codificados dentro do JWT. Mantidos mínimos de
// propósito: id e perfil, o suficiente para o middleware autorizar
// requisições sem precisar consultar o banco a cada request.
type Claims struct {
	UserID string      `json:"user_id"`
	Role   domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager encapsula a chave secreta e o tempo de expiração usados
// para assinar e validar tokens em toda a aplicação.
type JWTManager struct {
	secret     []byte
	expiryHrs  int
}

func NewJWTManager(secret string, expiryHrs int) *JWTManager {
	return &JWTManager{secret: []byte(secret), expiryHrs: expiryHrs}
}

// Generate cria um JWT assinado (HS256) para o usuário autenticado.
// Fluxo no sistema: chamado pelo AuthService após validar email/senha
// no login. Nenhum outro ponto do sistema deve emitir tokens.
func (m *JWTManager) Generate(user domain.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(m.expiryHrs) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse valida a assinatura e expiração de um token e devolve suas claims.
// Fluxo no sistema: chamado pelo middleware de autenticação em toda
// requisição protegida, antes de qualquer handler ser executado.
func (m *JWTManager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
