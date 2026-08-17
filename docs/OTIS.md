# Otis — Assistente de IA do ANT-Stock (V1)

Documentação técnica da primeira versão do Otis: assistente conversacional integrado ao ANT-Stock, rodando sobre Ollama/Qwen3 no VPS.

## 1. Visão geral do fluxo

```
Frontend (js/components/otis.js)
   ↓ POST /api/v1/otis/chat  (JWT no header, igual a qualquer outra rota)
Go API (OtisHandler)
   ↓
OtisService (internal/ia)
   ↓
OllamaClient (internal/ia)
   ↓ HTTP local
Ollama (127.0.0.1:11434)
   ↓
Qwen3 1.7B
   ↓
(resposta sobe pelo mesmo caminho até o frontend)
```

O navegador **nunca** acessa `127.0.0.1:11434` diretamente — só o backend Go fala com o Ollama.

## 2. Arquivos criados

| Arquivo | Responsabilidade |
|---|---|
| `backend/internal/ia/ollama.go` | Cliente HTTP para a API `/api/chat` do Ollama. Único ponto do sistema que monta essa requisição. |
| `backend/internal/ia/service.go` | `OtisService`: valida mensagem/histórico, monta o prompt final e delega ao `OllamaClient`. Define a interface `Retriever` (RAG futuro, não implementada nesta V1). |
| `backend/internal/ia/prompt.go` | System prompt do Otis, isolado do resto do código. `BuildSystemPrompt(context)` já aceita um bloco de contexto adicional para quando o RAG existir. |
| `backend/internal/handlers/ia_handler.go` | `OtisHandler.Chat` — tradução HTTP ↔ `OtisService`, no mesmo padrão de `SupportHandler`. |
| `frontend/.../js/components/otis.js` | Componente da bolha + painel de chat. Auto-montado uma única vez, observa a sessão para aparecer/sumir. |
| `frontend/.../css/otis.css` | Estilos do Otis, construídos inteiramente sobre os tokens já existentes em `variables.css`. |
| `docs/OTIS.md` | Este documento. |

## 3. Arquivos modificados

| Arquivo | Mudança |
|---|---|
| `backend/config/config.go` | Novas variáveis: `OLLAMA_BASE_URL`, `OLLAMA_MODEL`, `OLLAMA_TIMEOUT_SECS` (com defaults seguros para dev local). |
| `backend/routes/routes.go` | Grupo `/otis` (autenticado, sem restrição de perfil) com `POST /chat`. |
| `backend/cmd/main.go` | Instancia `OllamaClient` e `OtisService`, injeta `OtisHandler` em `routes.Dependencies`. |
| `backend/.env.example` | Documenta as três variáveis novas. |
| `frontend/.../js/services/api.js` | `API.otisChat(message, history)`. |
| `frontend/.../js/app.js` | Importa `otis.js` (auto-montagem). |
| `frontend/.../index.html` | `<link>` para `css/otis.css`. |

Nenhum outro arquivo foi tocado. Nenhuma migration nova foi criada — a V1 não persiste conversa (ver seção 6).

## 4. Endpoint

```
POST /api/v1/otis/chat
Authorization: Bearer <jwt>   (obrigatório — mesmo middleware RequireAuth de toda a API)

Body:
{
  "message": "Como cadastro um produto?",
  "history": [
    { "role": "user", "content": "..." },
    { "role": "assistant", "content": "..." }
  ]
}

Resposta 200:
{ "response": "Para cadastrar um produto..." }

Erros:
400 — mensagem vazia, muito longa (> 2000 caracteres) ou histórico muito longo (> 20 turnos)
401 — token ausente/inválido (aplicado pelo middleware antes do handler)
503 — Ollama indisponível ou erro de comunicação
```

Disponível para qualquer usuário autenticado (professor ou gestão) — não há restrição de perfil, pois o Otis nesta V1 só orienta sobre o uso do sistema e não expõe nem altera dados.

## 5. Segurança

- Autenticação: idêntica ao resto da API (`middleware.RequireAuth`). Sem token válido, a requisição nem chega ao `OtisHandler`.
- O Otis **não tem acesso ao banco** nesta V1. O system prompt (`prompt.go`) instrui explicitamente o modelo a não inventar dados de estoque.
- O Otis **não executa ações nem altera dados**. Não há function-calling/tools habilitados nesta versão — qualquer menção a "tools" no código é só a interface `Retriever`, que hoje não é usada (`nil` em `main.go`).
- Nenhuma URL ou credencial fica hardcoded: tudo vem de `config.Config`, carregado uma única vez em `main.go`.

## 6. Conversa temporária (sem persistência)

A conversa vive inteiramente em memória no módulo `otis.js` (array `messages`) enquanto o usuário usa o sistema. Nada é salvo em `localStorage`/`sessionStorage` nem em banco. Ao trocar de página o estado permanece (o componente não é remontado pelo router — ver seção 7), mas um refresh da página ou logout limpa tudo. O botão "limpar conversa" (ícone de borracha no cabeçalho do painel) zera o array a qualquer momento.

Isso deixa a porta aberta para persistência futura: bastaria trocar `messages = []`/`push` por chamadas a um novo endpoint de histórico, sem mudar a UI.

## 7. Decisões de arquitetura no frontend

O router do ANT-Stock remonta `#app` inteiro a cada navegação (`router.js` → `mount()`), então um widget persistente não pode viver dentro do conteúdo de uma rota. O projeto já tinha esse problema resolvido para as notificações (`#notifications`, montado direto no `index.html`, fora do `#app`) — o Otis segue exatamente o mesmo padrão: um `<div class="otis-root">` anexado a `document.body`, com um módulo (`otis.js`) que se inscreve em `session.subscribe(...)` e monta/desmonta sozinho no login/logout. Nenhuma página precisa saber que o Otis existe.

O ícone de estrela foi desenhado como um `<path>` SVG próprio (não é um ícone Lucide da biblioteca já usada no projeto), para ter controle total do formato sem depender de nenhuma logomarca de terceiros.

## 8. Preparado para RAG (Supabase + pgvector)

**Arquitetura planejada (ambiente principal):**

```
ANT-Stock → Go → Supabase PostgreSQL
                     ├── dados da aplicação (como hoje)
                     └── pgvector → embeddings da documentação do ANT-Stock
                                        ↓
                                  busca semântica
                                        ↓
                              contexto relevante → Qwen3 → Otis
```

O que já está pronto no código para isso:

- `ia.Retriever`: interface com um único método, `Retrieve(ctx, query) (string, error)`. Um `RetrievalService` futuro (embeddings + busca por similaridade no pgvector do Supabase) só precisa implementar essa interface.
- `OtisService` já recebe um `Retriever` no construtor (`nil` nesta V1) e já chama `Retrieve` antes de montar o prompt, ignorando erros de recuperação sem derrubar a conversa (o Otis responde normalmente, só sem o contexto extra).
- `prompt.BuildSystemPrompt(contexto)` já aceita o texto recuperado e o anexa ao prompt base — nenhuma mudança de contrato será necessária no dia em que o RAG entrar.
- Nenhum texto de documentação está hardcoded no código: a camada de IA foi mantida deliberadamente vazia de conteúdo específico do domínio além do prompt de persona.

**O que falta implementar quando o RAG for priorizado:**
1. Migration nova habilitando a extensão `pgvector` no Supabase e criando a tabela de chunks/embeddings.
2. Pipeline de ingestão da documentação (chunking + geração de embeddings).
3. `RetrievalService` (novo pacote `internal/rag` ou dentro de `internal/ia`, a definir na hora) implementando `ia.Retriever`.
4. Passar essa implementação em `main.go` no lugar do `nil` atual.

### 8.1 Plano de contingência — pgvector no VPS

O Supabase PostgreSQL é o banco principal da aplicação e será também o banco vetorial em produção normal. Como estratégia de resiliência, documenta-se aqui a alternativa de rodar PostgreSQL + pgvector diretamente no VPS, **caso seja necessário no futuro** — sem ativação nesta etapa.

```
Produção normal:      ANT-Stock → Supabase PostgreSQL + pgvector
Plano de contingência: ANT-Stock → PostgreSQL/pgvector hospedado no VPS
```

Procedimento técnico para ativar a alternativa, se um dia for necessário:

1. **Instalar o PostgreSQL no VPS** (versão compatível com a extensão pgvector, ex. PostgreSQL 15+):
   ```bash
   sudo apt update && sudo apt install postgresql postgresql-contrib
   ```
2. **Instalar/habilitar a extensão pgvector:**
   ```bash
   sudo apt install postgresql-15-pgvector   # ou compilar a partir do source, conforme a versão
   ```
   No banco de destino: `CREATE EXTENSION IF NOT EXISTS vector;`
3. **Configurar o banco:** criar usuário/role dedicado, database, e replicar o schema das tabelas vetoriais criadas originalmente no Supabase (mesma definição de colunas, índice `ivfflat`/`hnsw` conforme o volume de dados).
4. **Variáveis de ambiente:** o backend já lê toda configuração de banco via `config.Config` (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`). Apontar essas variáveis para o Postgres do VPS é suficiente — não é necessário reescrever `database.Connect` nem os repositories, já que o acesso é via `lib/pq`/SQL padrão, sem uso de SDK específico do Supabase.
5. **Migração dos embeddings:** exportar as tabelas vetoriais do Supabase (`pg_dump` filtrando pelas tabelas relevantes) e restaurar no Postgres do VPS.
6. **Alterar a conexão da aplicação:** trocar `DB_HOST`/credenciais no `.env` do backend e reiniciar o serviço.
7. **Procedimento para voltar ao Supabase:** processo inverso — `pg_dump` do VPS, restaurar no Supabase, reverter as variáveis de ambiente.

Importante:
- Supabase é o ambiente principal; pgvector no VPS é **só contingência**.
- As duas alternativas **não devem rodar simultaneamente** sem uma estratégia explícita de sincronização (não implementada nesta V1).
- Como toda configuração de banco já passa por `config.Config` e todo acesso já é feito via SQL padrão (sem SDK proprietário do Supabase no código do backend), migrar entre as duas opções não exige reescrever o sistema — só trocar variáveis de ambiente e replicar o schema.

## 9. Preparado para Tools

Nenhuma tool está implementada ou exposta ao modelo nesta V1 — o Qwen3 não recebe nenhuma definição de function-calling. A arquitetura já separa claramente onde as tools futuras (`consultar_estoque`, `consultar_produto`, `consultar_lote`, `consultar_movimentacoes`, `importar_produtos_excel`) vão entrar: como métodos chamados pelo `OtisService` (nunca pelo handler, nunca diretamente pelo modelo), cada um delegando para os services/repositories já existentes do domínio (`InventoryService`, `PreProductService`, etc.) — nunca com acesso direto ao PostgreSQL. Isso mantém a regra de que toda ação passa por validação de permissões no Go antes de tocar o banco.

## 10. Como testar localmente

### 10.1 Instalar/configurar o Ollama no VPS (ou máquina local de dev)

```bash
# instalação (se ainda não estiver instalado)
curl -fsSL https://ollama.com/install.sh | sh

# baixar o modelo usado pelo Otis
ollama pull qwen3:1.7b

# subir o serviço (geralmente já roda como serviço systemd após a instalação)
ollama serve
```

Verifique que está no ar:
```bash
curl http://127.0.0.1:11434/api/tags
```

### 10.2 Configurar o backend

No `backend/.env` (copiado de `.env.example`), garanta:
```
OLLAMA_BASE_URL=http://127.0.0.1:11434
OLLAMA_MODEL=qwen3:1.7b
OLLAMA_TIMEOUT_SECS=60
```

Suba o backend normalmente:
```bash
cd backend
go run ./cmd/main.go
```

### 10.3 Testar a comunicação Go → Ollama isoladamente

```bash
curl -X POST http://localhost:8000/api/v1/otis/chat \
  -H "Authorization: Bearer <SEU_JWT_DE_LOGIN>" \
  -H "Content-Type: application/json" \
  -d '{"message": "Como cadastro um produto no ANT-Stock?", "history": []}'
```
Uma resposta 200 com `{"response": "..."}` confirma o fluxo completo Go → Ollama → Qwen3 → Go.

### 10.4 Testar no frontend

Sirva o frontend como já é feito hoje (sem build step) e faça login normalmente. A bolha com a estrela aparece no canto inferior direito assim que a sessão estiver autenticada; clique para abrir o painel, envie uma mensagem e confirme que a resposta chega, que o Enter envia, que "limpar conversa" funciona e que desconectar a rede (ou derrubar o Ollama) mostra a mensagem de erro no painel sem quebrar o restante da aplicação.

## 11. Dependências

Nenhuma dependência nova de Go ou npm foi adicionada — `net/http` e `encoding/json` (biblioteca padrão) bastam para o `OllamaClient`. A única instalação necessária é a do próprio **Ollama** no ambiente onde o backend roda, com o modelo `qwen3:1.7b` baixado (`ollama pull qwen3:1.7b`), conforme seção 10.1. Isso deve ser feito diretamente no VPS.

## 12. Decisões arquiteturais relevantes

- O pacote `internal/ia` já existia como esqueleto vazio no projeto (`ollama.go`, `service.go`, `prompt.go` sem conteúdo) — foi usado como estava planejado, sem criar uma estrutura paralela.
- `ia.OtisService` não é um `service` dentro de `internal/services` porque não segue o mesmo contrato dos demais (não recebe repository, não faz acesso a banco, tem cliente HTTP próprio com timeout e ciclo de vida distintos). Manter em `internal/ia` evita misturar dois tipos de dependência diferentes na mesma pasta.
- A rota `/otis/chat` foi liberada para qualquer usuário autenticado (sem `RequireRole`), diferente de rotas administrativas, porque o Otis nesta V1 é só orientação de uso — não há dado sensível envolvido.
- O histórico da conversa é enviado pelo frontend a cada requisição (stateless no backend) em vez de guardado em sessão no servidor, porque a V1 não deveria introduzir estado de conversa no backend antes de existir uma decisão sobre persistência real — trocar isso por histórico persistido no futuro é aditivo, não uma ruptura de contrato.
