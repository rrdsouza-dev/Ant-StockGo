# WMS Backend (Go)

API REST em Go que substitui completamente o backend anterior em Python.
Arquitetura: `handlers -> services -> repositories -> PostgreSQL (Supabase)`.

## Requisitos

- Go 1.22+
- Uma instância PostgreSQL (recomendado: [Supabase](https://supabase.com))

## 1. Configurar variáveis de ambiente

```bash
cp .env.example .env
# edite .env com as credenciais do seu banco Supabase/Postgres e um JWT_SECRET forte
```

## 2. Criar o schema no banco

Cole o conteúdo de `database/migrations/001_init.sql` no SQL Editor do
Supabase, ou rode via psql:

```bash
psql "$DATABASE_URL" -f database/migrations/001_init.sql
```

Isso cria as tabelas `users`, `pending_users`, `deposits`, `inventory`,
`stock_movements`, `classes`, `class_teachers` e `class_deposits`.

> **Primeiro acesso:** como toda conta nova entra como pendente, não há
> nenhum usuário de gestão para aprovar os demais. Descomente e ajuste o
> INSERT de seed no final do arquivo de migração (ou gere um hash bcrypt
> próprio) para criar a primeira conta de gestão diretamente no banco.

## 3. Instalar dependências

```bash
go mod init wms-backend   # pule se o go.mod já existir (já está incluído neste projeto)
go get github.com/gin-gonic/gin
go get github.com/golang-jwt/jwt/v5
go get github.com/joho/godotenv
go get golang.org/x/crypto/bcrypt
go get github.com/lib/pq
go mod tidy
```

## 4. Rodar em desenvolvimento

```bash
go run main.go
```

O servidor sobe em `http://localhost:8000` (ou na porta definida em `PORT`).
Verifique com: `curl http://localhost:8000/health`

## 5. Build para produção

```bash
go build -o wms-backend
./wms-backend
```

## 6. Docker (opcional)

```bash
docker build -t wms-backend .
docker run --env-file .env -p 8000:8000 wms-backend
```

## Endpoints

Ver `routes/routes.go` para a lista completa. Resumo:

| Método | Rota                    | Acesso           |
|--------|--------------------------|-------------------|
| POST   | `/api/v1/auth/login`     | Público           |
| POST   | `/api/v1/auth/register`  | Público (cria pendente) |
| GET    | `/api/v1/auth/pending`   | Gestão            |
| POST   | `/api/v1/auth/approve`   | Gestão            |
| GET    | `/api/v1/users/me`       | Autenticado       |
| GET    | `/api/v1/users`          | Gestão            |
| GET/POST/PATCH/DELETE | `/api/v1/deposits[/:id]` | Leitura: autenticado · Escrita: gestão |
| GET/POST/PATCH/DELETE | `/api/v1/inventory[/:id]` | Leitura: autenticado · Escrita: gestão |
| POST   | `/api/v1/inventory/move` | Autenticado (escopo por depósito) |
| GET    | `/api/v1/inventory/movements` | Autenticado |
| GET/POST/PATCH/DELETE | `/api/v1/classes[/:id]` | Leitura: autenticado · Escrita: gestão |
