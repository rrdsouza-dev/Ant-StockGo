# ANT Stock — Front-End

Front-end estático em HTML, CSS e JavaScript puro (ES Modules) para o WMS
ANT Stock. Não há React, Vite, JSX, TSX ou etapa de build.

## Stack

- HTML5 + CSS3
- JavaScript ES Modules
- Lucide Icons, Chart.js e SheetJS via CDN
- Roteador hash em `js/router.js`
- Cliente HTTP para o backend Go em `js/services/api.js`

## Integração com o backend

Por padrão, o front consome `http://localhost:8000/api/v1` (backend Go,
ver pasta `backend/`). Para trocar a URL sem build, defina
`window.ANT_API_BASE_URL` antes de carregar `js/app.js`, por exemplo:

```html
<script>window.ANT_API_BASE_URL = "https://api.minhaescola.com/api/v1";</script>
```

Endpoints consumidos (ver `js/services/api.js`):

- `POST /auth/login`, `POST /auth/register`
- `GET /auth/pending`, `POST /auth/approve` (gestão)
- `GET /users/me`, `GET /users` (gestão)
- `GET /deposits` (`?scope=stock&class_id=` para acesso a estoque), `POST/PATCH/DELETE /deposits[/:id]` (gestão)
- `GET/POST/PATCH/DELETE /inventory[/:id]` (aceita `?class_id=`), `POST /inventory/move`, `GET /inventory/movements`
- `GET/POST/PATCH/DELETE /classes[/:id]`
- `GET/POST /categories`
- `POST /support/tickets` (professor), `GET /support/tickets` (gestão), `DELETE /support/tickets` (gestão + código administrativo)

## Estrutura de páginas

| Rota | Página | Acesso |
|---|---|---|
| `/dashboard` | Visão geral (estoque + movimentações) | Todos |
| `/inventory` | Estoque (itens de inventário) — CRUD completo | Todos, restrito ao(s) depósito(s) acessível(is) |
| `/movements` | Entradas/Saídas + leitor de código de barras | Todos |
| `/classes` | Turmas (vínculo professor↔turma por checkbox) | Todos (edição: gestão) |
| `/deposits` | Depósitos (inclui marcar o depósito administrativo) | Todos (edição: gestão) |
| `/reports` | Histórico de movimentações | Todos |
| `/users` | Aprovação de contas + usuários ativos | Gestão |
| `/support` | Suporte (professor abre chamado; gestão visualiza/exporta/limpa) | Todos |
| `/choose-class` | "Qual turma deseja acessar?" — escolha/troca de turma ativa | Professor com mais de uma turma |
| `/exports` | Exportação de dados (TXT/Excel) | Todos |
| `/settings`, `/profile` | Preferências e perfil (inclui código de suporte) | Todos |

### Modelo de acesso a estoque

Gestão só acessa o estoque do depósito marcado como administrativo;
professor acessa os depósitos da turma ativa (escolhida em
`/choose-class`, guardada em `session.classId`) ou a união de todas as
suas turmas. Essa regra é sempre revalidada no backend — o frontend só
envia `class_id` como sugestão, nunca como autorização.

## Executar

Sirva esta pasta com qualquer servidor estático. Exemplo:

```bash
python3 -m http.server 5173 --directory public/frontend
# ou
npx serve public/frontend
```

Suba o backend Go (ver `backend/README.md`) antes de fazer login — o
front não funciona sem a API respondendo em `ANT_API_BASE_URL`.
