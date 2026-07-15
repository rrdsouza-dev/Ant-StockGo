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
psql "$DATABASE_URL" -f database/migrations/002_item_details.sql
psql "$DATABASE_URL" -f database/migrations/003_admin_deposit_and_support.sql
```

`001_init.sql` cria as tabelas `users`, `pending_users`, `deposits`,
`inventory`, `stock_movements`, `classes`, `class_teachers` e
`class_deposits`. `002_item_details.sql` cria `categories` (com um seed
de categorias de exemplo) e adiciona ao `inventory` os campos de
validade, lote, categoria, localização e observações. `003_admin_deposit_and_support.sql`
adiciona `deposits.is_administrative` (com constraint garantindo no
máximo um depósito administrativo por vez), `users.support_code`, e cria
a tabela `support_tickets`. Todas seguras para rodar sobre um banco já
existente (`ADD COLUMN IF NOT EXISTS` / `CREATE TABLE IF NOT EXISTS`).

> **Primeiro acesso:** como toda conta nova entra como pendente, não há
> nenhum usuário de gestão para aprovar os demais. Descomente e ajuste o
> INSERT de seed no final do arquivo de migração (ou gere um hash bcrypt
> próprio) para criar a primeira conta de gestão diretamente no banco.
>
> **Depósito administrativo:** após ter uma conta de gestão, crie ou
> edite um depósito marcando o checkbox "Depósito administrativo" — é o
> único depósito cujo estoque a gestão poderá acessar (as demais telas de
> estoque ficam vazias para ela até isso ser feito).

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
| GET    | `/api/v1/users/me`       | Autenticado (inclui `support_code` para professores) |
| GET    | `/api/v1/users`          | Gestão            |
| GET `/api/v1/deposits`  | (padrão) todos para gestão, das turmas para professor · `?scope=stock&class_id=` acesso ao ESTOQUE (gestão: só o depósito administrativo; professor: turma informada ou união de todas) | Autenticado |
| POST/PATCH/DELETE `/api/v1/deposits[/:id]` | Gestão |
| GET/POST/PATCH/DELETE | `/api/v1/inventory[/:id]` (aceita `?class_id=` na leitura) | Autenticado, escopo por depósito (ver seção de permissões abaixo) |
| POST   | `/api/v1/inventory/move` | Autenticado (escopo por depósito) |
| GET    | `/api/v1/inventory/movements` (aceita `?class_id=`) | Autenticado |
| GET/POST/PATCH/DELETE | `/api/v1/classes[/:id]` | Leitura: autenticado · Escrita: gestão |
| GET/POST | `/api/v1/categories` | Autenticado (gestão e professor) |
| POST   | `/api/v1/support/tickets` | Professor |
| GET    | `/api/v1/support/tickets` | Gestão |
| DELETE | `/api/v1/support/tickets` | Gestão (exige `admin_code` no corpo, validado contra `SUPPORT_ADMIN_CODE`) |

`POST`/`PATCH /inventory` aceitam, além de nome/sku/quantidade mínima:
`expiry_date` (string `"DD/MM/AAAA"`, obrigatório), `lot_number`,
`category_id` (uuid ou `null`), `notes`, e `location` (objeto
`{aisle, tower, shelf, position}`, todos os campos opcionais e
independentes — pensado para crescer sem precisar de nova migração).

## Permissões de estoque (gestão vs. professor)

- **Gestão**: continua criando/editando/excluindo depósitos e turmas
  como entidades, e aprovando contas. Acesso ao **estoque** (itens e
  movimentações), porém, fica restrito ao depósito marcado como
  administrativo (`deposits.is_administrative`).
- **Professor**: CRUD completo de itens de estoque e movimentações,
  sempre restrito aos depósitos das turmas às quais está vinculado (ou
  apenas à turma escolhida na sessão, via `?class_id=`).

Toda essa regra vive em `ClassService.AccessibleDepositIDs` — nenhuma
outra camada decide isso de forma independente.

## Variáveis de ambiente adicionais

- `SUPPORT_ADMIN_CODE`: código exigido da gestão para limpar o histórico
  de chamados de suporte. Sem valor definido, a limpeza fica sempre
  bloqueada (nunca aceita código vazio como válido).
