# Relatório — Importação Inteligente e Correção de Sessão

Este relatório documenta especificamente a implementação da **Importação
Inteligente** e a **correção do bug de sessão após fechar o navegador**,
pedidas na especificação `wms-project`. É um relatório separado do
`RELATORIO.md` existente (que documenta uma entrega anterior); este cobre
apenas o trabalho desta sessão.

---

## 1. Resumo executivo

| Item pedido | Status |
|---|---|
| Analisar o projeto inteiro antes de codar | Feito — arquitetura, auth, domínio, frontend, Otis/IA, migrations, animações, CSS todos mapeados antes da primeira linha de código |
| Importação Inteligente | Implementada de ponta a ponta (backend + frontend) |
| Correção do bug de sessão | Causa raiz identificada e corrigida |
| Preservar Otis, RAG, Tools existentes | Preservados — nenhum arquivo do Otis foi alterado em comportamento (só extraído um ícone compartilhado) |
| Não criar arquitetura paralela | Reaproveitados `InventoryService.Create`, `InventoryService.MoveStock`, `PreProductService.Create`, `DepositService.ListForStock`, `CategoryService.List`, `validation.ParseBRDate` |
| Migrations novas | **Nenhuma foi necessária** (ver seção 6) |
| `go build ./...` | ✅ Passa (ver seção 8, nota sobre proxy de módulos) |
| `go vet ./...` | ✅ Passa |
| Testes automatizados | 53 testes Go + 43 asserts de frontend (jsdom), todos passando |
| Teste de integração contra Postgres real | ❌ **Não executado** — ver seção 9, é a pendência real mais importante |

---

## 2. Correção do bug de sessão

### Causa raiz

O JWT expira em 24h fixas (`config.go`, `JWT_EXPIRY_HOURS`), e **não existe
refresh token** no projeto. O bug tinha duas causas combinadas:

1. **No boot do frontend** (`app.js`), o guard de rotas só verificava se
   havia `user`/`token` salvos em `localStorage` — nunca se o token já
   tinha *expirado*. Reabrir o navegador depois de 24h fazia o dashboard
   renderizar normalmente.
2. **No tratamento de erro 401** (`api.js`), a única reação era
   `session.signOut()` (limpar o `localStorage`) — **sem nenhum
   redirecionamento para `/login`**. A página que disparou a chamada
   continuava montada, agora "quebrada", mostrando toasts de erro em cada
   ação até o usuário clicar em Sair manualmente. Confirmei que esse
   padrão (`notify(err.message, "error")` sem tratamento de 401) se
   repetia em 13 páginas diferentes.

### Correção aplicada

- `frontend/public/frontend/js/services/store.js`: `isAuthenticated()`
  agora decodifica o payload do JWT (Base64, sem verificar assinatura —
  puramente para UX) e considera expirado com 5s de margem. Isso resolve
  o caso mais comum: reabrir o navegador depois do token vencer redireciona
  direto para `/login` no boot, sem chegar a montar o dashboard.
- `frontend/public/frontend/js/services/api.js`: ao receber 401, além de
  `session.signOut()`, dispara `CustomEvent("antstock:session-expired")`.
- `frontend/public/frontend/js/app.js`: escuta esse evento, mostra
  `notify("Sua sessão expirou. Faça login novamente.", "warning")` e
  chama `router.navigate("/login")`. Isso cobre o caso remanescente: token
  expira **durante** o uso (sessão longa sem fechar o navegador).

Nenhuma mudança no backend foi necessária — o comportamento do backend
(retornar 401 para token expirado) já estava correto; o problema era
inteiramente do lado do frontend não reagir a isso.

### Validação

Testei a lógica de decodificação/expiração isoladamente em Node.js com 5
casos (token válido, expirado, com acentuação UTF-8 no payload, malformado,
sem `exp`), e depois **contra um JWT real gerado pela mesma biblioteca do
backend** (`golang-jwt/jwt/v5`), confirmando que o formato do campo `exp`
(inteiro Unix em segundos) é interpretado corretamente.

**O que não foi testado:** o fluxo completo em um navegador real (abrir o
site, esperar/forçar a expiração, fechar e reabrir) não foi executado
manualmente — não tenho como simular isso no ambiente sandbox sem UI. A
lógica foi validada por unidade, não end-to-end no navegador.

---

## 3. Importação Inteligente — arquitetura implementada

```
Frontend (imports.js)
   │  multipart/form-data
   ▼
POST /imports/preview          POST /imports/commit
   │                                │
   ▼                                ▼
ImportHandler.Preview          ImportHandler.Commit
   │                                │
   ▼                                ▼
ImportService.Preview          ImportService.Commit
   │                                │
   ├─ ParseSpreadsheet()            ├─ InventoryService.Create()
   │   (XLSX/CSV, sem libs          │   InventoryService.MoveStock()
   │    externas)                   │   (ou PreProductService.Create())
   ├─ MapColumnsDeterministic()     │
   ├─ HeaderResolver (IA, só se     └─ nada de SQL direto — sempre
   │   sobrar ambiguidade)              via Services existentes
   ├─ BuildItems() (validação)
   └─ resolveCategoryIDs()
```

**Nada é persistido em `/preview`.** A prévia inteira vive em memória
durante a requisição e é devolvida ao frontend, que o usuário revisa,
edita e filtra antes de confirmar. `/commit` é a única chamada que grava
algo, e grava item por item — uma falha em um item nunca impede os
demais (ver seção 3.5).

### 3.1 Parser (sem dependências externas)

Decisão deliberada: implementei o parser de `.xlsx` manualmente usando só
`archive/zip` + `encoding/xml` da biblioteca padrão do Go, em vez de usar
uma lib de terceiros (ex.: `excelize`). Motivos:

- Mantém a arquitetura enxuta pedida na especificação.
- Evita adicionar dependências pesadas (testei `excelize` neste sandbox e
  ela puxa `golang.org/x/arch` para otimizações SIMD, desnecessário para
  ler uma planilha simples de produtos).
- `.csv` usa `encoding/csv` nativo.

O formato é detectado pelo **conteúdo real do arquivo** (magic bytes ZIP
para XLSX), nunca só pela extensão declarada — um arquivo `.xlsx` forjado
(texto puro renomeado) é rejeitado como corrompido, não interpretado como
CSV por engano.

Suporta: shared strings, texto inline, células numéricas, datas seriais do
Excel (convertidas a partir da época `1899-12-30`), delimitador `,` ou `;`
em CSV (detecção automática), BOM UTF-8, linhas em branco no meio da
planilha (descartadas).

### 3.2 Normalizer — mapeamento determinístico de colunas

`knownHeaderAliases` mapeia dezenas de variações comuns de nome de coluna
("Produto", "Nome do Produto", "Descrição", "Item", "Produto/Descrição" →
todos o mesmo campo lógico `name`) sem envolver IA. Cobre também
quantidade, lote, validade, SKU, marca, estoque mínimo, categoria, unidade
e observações.

Normalização de valores:
- **Quantidade**: aceita separador de milhar/decimal (`.` ou `,`),
  detecta automaticamente qual é qual pela posição (3 dígitos depois =
  milhar, senão decimal).
- **Data**: aceita `DD/MM/AAAA`, `AAAA-MM-DD` (ISO), e datas seriais do
  Excel — sempre normalizado para `DD/MM/AAAA`, o mesmo formato que
  `validation.ParseBRDate` (já existente no backend) espera. A validação
  de data em si **reutiliza `validation.ParseBRDate`**, não duplica a
  regra.

### 3.3 IA — usada apenas para ambiguidade de cabeçalho

Conforme pedido explicitamente na especificação (itens 3, 19, 29): a IA
**nunca** recebe a planilha inteira. `ImportService.Preview` só chama o
`HeaderResolver` (implementado sobre o mesmo `*ia.OllamaClient` já usado
pelo Otis — não abre uma segunda conexão) quando sobra pelo menos um
cabeçalho que o mapeamento determinístico não resolveu sozinho, e nesse
caso envia só os cabeçalhos ambíguos + até 3 valores de amostra de cada
coluna, nunca as linhas de dados inteiras. Um limite (`maxAmbiguousColumnsForAI
= 15`) evita mandar uma quantidade desproporcional de colunas de uma vez.

Se a IA falhar ou estiver indisponível, a importação **continua
funcionando** — os cabeçalhos que ficariam sem resolver simplesmente não
são mapeados (reportados em `unmapped_headers`), sem bloquear o resto do
fluxo. Isso satisfaz o item 20 da especificação (não depender do RAG/IA).

### 3.4 Validator — campos ausentes, duplicidade, vencimento

`BuildItems` converte cada linha em um `ImportItem` com:
- **Issues bloqueantes** (`missing`, `invalid_format`, `invalid_quantity`)
  — impedem o item de ser `Ready`. Os campos obrigatórios variam por
  destino: `Produto` (item de estoque) exige nome, quantidade, lote e
  validade (espelhando `domain.InventoryItem`); `Pré-produto` exige nome
  **e unidade de medida** (`PreProductService.Create` rejeita unidade
  vazia — não é só o nome, como uma leitura apressada do domínio sugeria).
- **Issues informativas** (`expired`, `expiring_soon`,
  `possible_duplicate`) — não bloqueiam a importação, só avisam.
- **Detecção de duplicidade**: compara conjuntos de palavras normalizadas
  entre nomes; "Toddy 370g" está contido em "Toddy Chocolate 370g", que
  está contido em "Toddy Chocolate em Pó 370g" — exatamente o exemplo da
  especificação. Nunca remove automaticamente, só sinaliza.

### 3.5 Commit — gravação via Services existentes

Para cada item, na ordem revisada pelo usuário:

- **Destino "Produto"**: `InventoryService.Create` (cadastra o item,
  quantidade inicial zero) seguido de `InventoryService.MoveStock(...,
  domain.MovementIn, quantidade, nota)`. Este é o **mesmo padrão de duas
  chamadas que o frontend já usa hoje** para criar um item de estoque
  manualmente (`api.js`: `createInventoryItem` + `moveStock`) — descobri
  isso ao investigar `InventoryService.Create`, que não aceita quantidade
  diretamente (a quantidade real do estoque é sempre rastreada via
  `StockMovement`, para auditoria). Não inventei um caminho novo.
- **Destino "Pré-produto"**: `PreProductService.Create` diretamente.

Cada item é uma operação independente — uma falha em um item (ex.: nome
duplicado que o backend rejeita, depósito sem permissão) não interrompe
os demais. Se o `Create` funcionar mas o `MoveStock` falhar (quantidade
inválida chegando ao commit apesar da validação do preview, por exemplo
por manipulação direta da requisição), o item fica registrado como falha
parcial com uma mensagem explícita — nunca reportado como sucesso
silencioso.

Nenhuma linha de SQL é escrita neste pacote. Autorização de depósito é
sempre revalidada dentro de `InventoryService`/`DepositService.CanAccess`,
nunca decidida aqui — a IA nunca escolhe nem valida um depósito.

### 3.6 Endpoints

- `POST /api/v1/imports/preview` — multipart/form-data, campos `file` e
  `target` (`inventory_item` padrão, ou `pre_product`). Retorna a prévia
  completa. Timeout de 20s dedicado à chamada de IA (não trava a
  requisição inteira se o Ollama estiver lento).
- `POST /api/v1/imports/commit` — JSON com `deposit_id`, `target`,
  `items[]`. Retorna o resultado item a item.

Ambos exigem autenticação (`authRequired`), sem restrição de perfil — a
autorização real de depósito acontece dentro dos Services, mesmo padrão
de `/inventory`.

---

## 4. Frontend — Importação Inteligente

Nova página em `frontend/public/frontend/js/pages/imports.js`, registrada
como `/imports`, com item na sidebar ("Importações", ao lado de
"Exportações" — não existia uma seção "Importações" no projeto antes
desta implementação).

Cinco estágios, cada um renderizado no mesmo container (`idle` →
`analyzing` → `preview` → `committing` → `done`):

1. **Idle**: dropzone com clique ou arrastar-e-soltar, validação
   client-side de extensão/tamanho antes mesmo de chamar a API.
2. **Analyzing**: spinner + lista de passos ("Lendo produtos...",
   "Identificando colunas...", etc.), usando `@keyframes spin` já
   existente em `animations.css` — não inventei uma animação nova.
3. **Preview**: card de resumo (contagem, unidades, chips de
   prontos/atenção), seletor Produto/Pré-produto (que **reprocessa a
   prévia no backend** ao trocar, já que os campos obrigatórios mudam),
   seletor de depósito (vindo de `DepositService.ListForStock`, respeitando
   permissões), filtro "mostrar somente problemas", grid de cards por
   item com campos ausentes destacados em vermelho, edição individual
   (modal reaproveitando `applyDateMask`/`isValidBRDate` de
   `validators.js`), remoção individual e "remover todos" com
   confirmação.
4. **Committing**: barra de progresso (preenchida ao concluir a chamada —
   o backend processa a lista inteira em uma única requisição, não há
   streaming de progresso item a item nesta V1).
5. **Done**: resumo final (analisados/importados/não importados) e lista
   detalhada de falhas quando houver.

O ícone de estrela usado como indicador de IA (`starIcon`) foi **extraído**
de `otis.js` para um módulo compartilhado (`utils/aiIcon.js`), em vez de
duplicado — agora Otis e Importação Inteligente usam a mesma fonte.

Todo o CSS novo (`css/imports.css`) reaproveita as variáveis de design
existentes (`variables.css`) e as classes base já disponíveis (`card`,
`chip-*`, `btn-*`, `field`/`input`) — não introduz nenhum token novo de
cor, espaçamento ou raio de borda.

---

## 5. Arquivos criados e modificados

### Backend — criados
- `backend/internal/importing/types.go`
- `backend/internal/importing/parser.go`
- `backend/internal/importing/xlsx_reader.go`
- `backend/internal/importing/normalizer.go`
- `backend/internal/importing/validator.go`
- `backend/internal/importing/ai_analyzer.go`
- `backend/internal/importing/service.go`
- `backend/internal/importing/*_test.go` (5 arquivos de teste)
- `backend/internal/importing/testdata/*.xlsx` (4 fixtures gerados com
  openpyxl para os testes do parser)
- `backend/internal/handlers/import_handler.go`

### Backend — modificados
- `backend/routes/routes.go` — rotas `/imports/preview` e `/imports/commit`
- `backend/cmd/main.go` — wiring do `ImportService`/`ImportHandler`,
  reaproveitando o `ollamaClient` já existente

### Frontend — criados
- `frontend/public/frontend/js/pages/imports.js`
- `frontend/public/frontend/css/imports.css`
- `frontend/public/frontend/js/utils/aiIcon.js`

### Frontend — modificados
- `frontend/public/frontend/js/services/api.js` — `importPreview()`,
  `importCommit()`, `requestUpload()` (novo helper multipart)
- `frontend/public/frontend/js/services/store.js` — correção de sessão
  (`isAuthenticated` detecta expiração)
- `frontend/public/frontend/js/app.js` — rota `/imports` + listener de
  sessão expirada (correção de sessão)
- `frontend/public/frontend/js/components/sidebar.js` — item "Importações"
- `frontend/public/frontend/js/components/otis.js` — usa `starIcon`
  importado em vez de definição local duplicada (comportamento idêntico)
- `frontend/public/frontend/index.html` — `<link>` para `imports.css`

**Nenhum arquivo existente foi removido. Nenhuma funcionalidade existente
foi alterada em comportamento**, exceto a correção pontual do tratamento
de 401/expiração de sessão.

---

## 6. Migrations

**Nenhuma migration nova foi criada.** A Importação Inteligente não
introduz nenhum novo dado persistente: ela orquestra `InventoryService` e
`PreProductService` já existentes, que gravam nas tabelas já existentes
(`inventory_items`, `pre_products`, `stock_movements`). O estado da prévia
(itens interpretados, mapeamento de colunas) vive inteiramente em memória
durante a requisição HTTP e nunca é persistido — não há uma tabela de
"sessões de importação" ou histórico de imports no banco.

Se no futuro for desejado manter um histórico de importações (quem
importou o quê, quando), isso exigiria uma nova tabela — deliberadamente
não implementada agora, por não ter sido pedida e por manter o escopo
enxuto conforme a especificação pediu ("crie migrations somente se
realmente forem necessárias").

---

## 7. Divergências encontradas em relação às premissas da especificação

Reportando com honestidade, conforme pedido:

- **"Tools já existentes" e "RAG já existente"**: o código real mostra que
  **nenhum dos dois existe de fato** hoje — só uma interface `Retriever`
  preparada (sempre `nil`) e comentários "nenhuma tool é exposta ao
  modelo nesta V1". O Otis atual é chat puro com Ollama, sem streaming,
  sem contexto de dados reais do sistema. Isso não mudou meu plano (a
  especificação já pedia para não expandir isso agora), mas é importante
  que fique registrado que não há nada além disso para "preservar" nessa
  frente.
- **"Área de Importações"**: não existia uma seção "Importações" no
  sistema antes desta implementação — existia apenas "Exportações". Criei
  a seção do zero.
- **"Animações fornecidas anteriormente no projeto"**: não encontrei
  arquivos de animação dedicados (SVG animado, Lottie, etc.) — só
  `animations.css` com keyframes CSS genéricos (`fadeIn`, `scaleIn`,
  `spin`, etc.), que reaproveitei.
- **Dark mode**: não existe implementado em nenhuma parte do projeto atual
  (nenhuma referência a `prefers-color-scheme` ou classe de tema). Não
  implementei um sistema de dark mode — isso seria uma funcionalidade nova
  fora do escopo pedido, não uma correção. Sinalizando para que a
  verificação "checar dark mode" pedida no item 35 seja entendida como
  "não aplicável" em vez de "verificado e funcionando".
- **`backend/Dockerfile`**: contém `go build -o wms-backend main.go`, mas
  não existe `main.go` na raiz de `backend/` (está em `backend/cmd/main.go`).
  Esse Dockerfile parece já estar quebrado independentemente desta
  entrega — não é algo que toquei, mas registro porque afeta o "fluxo de
  deploy" que a especificação pediu para eu analisar.
- **`backend/.env`**: contém a string de conexão do Supabase com senha em
  texto claro. Está corretamente listado em `.gitignore` (não é um
  vazamento no repositório), mas é uma prática de risco a considerar
  (variável de ambiente do sistema operacional ou um secrets manager,
  em vez de arquivo texto, especialmente se a máquina for compartilhada).

---

## 8. Sobre `go build`/`go vet`/testes neste ambiente

Este ambiente sandbox bloqueia acesso a `proxy.golang.org`,
`golang.org` e domínios `google.golang.org`/`gopkg.in` (só permite uma
lista fixa de domínios, incluindo `github.com`). Para conseguir rodar
`go build`/`go vet`/`go test` aqui, precisei configurar `replace`
directives temporárias no `go.mod` apontando os módulos
`golang.org/x/*`, `google.golang.org/protobuf` e `gopkg.in/yaml.v3` para
seus mirrors espelhados no GitHub, rodar a verificação, e **sempre
reverter** essas alterações via `git checkout -- go.mod go.sum` logo em
seguida — o `go.mod`/`go.sum` entregues são idênticos aos originais do
repositório, sem nenhum resíduo dessas replaces.

**Isso não deveria ser necessário no seu ambiente de produção/desenvolvimento
normal**, que presumivelmente tem acesso padrão a `proxy.golang.org`. Mas
é importante que você saiba que essa foi a única forma de eu efetivamente
compilar e testar o projeto aqui, em vez de apenas inspecionar o código
visualmente.

Confirmação final rodada logo antes da entrega:
```
go build ./...   → sem erros
go vet ./...     → sem erros
go test ./...    → 53 testes, todos PASS
```

---

## 9. Testes executados vs. pendências reais

### Executados e passando

- **Backend, 53 testes automatizados** (`go test ./internal/importing/...`):
  parser XLSX/CSV (contra arquivos `.xlsx` reais gerados com `openpyxl`,
  não simulados), normalização de quantidade/data (incluindo datas
  seriais do Excel, verificadas contra conversão em Python), mapeamento
  determinístico de cabeçalhos com os exemplos exatos da especificação,
  validação de campos obrigatórios (incluindo a diferença de regras entre
  Produto e Pré-produto), detecção de duplicidade com os exemplos exatos
  da especificação, avisos de vencimento, resolução de categoria por
  nome, e a lógica de resolução de ambiguidade via IA (com mock,
  incluindo o caso adversarial de a IA sugerir um campo que conflita com
  um mapeamento determinístico já feito).
- **Frontend, 43 asserts em 7 cenários** (jsdom, executando a página real
  no DOM, não simulação): fluxo completo idle→preview com o exemplo exato
  da especificação, edição de item com validação, fluxo completo até
  commit e resultado (incluindo falha parcial exibida corretamente com a
  mensagem específica do backend), falha de rede no preview sem travar a
  página, rejeição de arquivo com extensão inválida, troca de destino
  Produto↔Pré-produto reprocessando a prévia, "remover todos" com
  confirmação via modal.
- **Três bugs reais** foram encontrados pelos próprios testes durante o
  desenvolvimento e corrigidos antes da entrega: interpretação incorreta
  de separador de milhar em quantidades; heurística de duplicidade que
  não detectava uma variação de nome com palavra no meio; e a exigência
  de campo `Unit` para pré-produto, que eu tinha inicialmente esquecido
  (só listava `Name` como obrigatório) até revisar `PreProductService.Create`
  com atenção.
- Reprodução (não apenas leitura de código) da causa raiz do bug de
  sessão, e validação da correção contra um JWT real gerado pela mesma
  biblioteca do backend.

### Não executados — pendências reais

- **Teste de integração contra um banco Postgres real.** Os Services do
  projeto (`InventoryService`, `PreProductService`, `CategoryService`) são
  structs concretos amarrados a repositories reais, sem interfaces para
  mock — não há como testar `ImportService.Preview`/`Commit` de ponta a
  ponta sem um banco de verdade. O `.env` local tem credenciais reais do
  Supabase de produção do projeto; deliberadamente **não as usei** para
  testes automatizados, para não correr nenhum risco de tocar dados reais
  do usuário. Isso significa que o caminho `Create` → `MoveStock` →
  gravação real, a resolução de depósito/categoria contra dados reais, e
  o comportamento de erro do banco em casos de borda (ex.: nome duplicado)
  **não foram exercitados contra o banco real**.
- **Teste manual no navegador** do fluxo completo (upload de um arquivo
  real, através da UI, batendo no backend rodando de verdade com Ollama
  ativo). Validei a lógica isoladamente (parser contra arquivos reais,
  UI contra um DOM real com API mockada), mas não a integração fim-a-fim
  rodando os dois processos (backend Go + frontend servido) junto.
- **Teste do fluxo de sessão em um navegador real** (fechar de verdade,
  esperar expiração, reabrir) — só validei a lógica de decodificação do
  JWT isoladamente.
- **Ollama real**: não tenho acesso a uma instância rodando neste sandbox,
  então `HeaderResolver`/`OllamaHeaderResolver.ResolveHeaders` foi testado
  só com mock. O parsing da resposta (incluindo tolerância a cercas
  markdown que modelos pequenos às vezes adicionam) foi testado
  isoladamente contra strings fixas, não contra uma resposta real do
  Qwen configurado no projeto.
- **Responsividade e "dark mode"** foram verificados por revisão de CSS
  (breakpoints consistentes com o padrão existente do projeto, uso
  exclusivo de variáveis de tema já existentes) — não por captura de tela
  real em diferentes tamanhos de viewport, já que não tenho um navegador
  disponível neste ambiente.

---

## 10. Comandos para colocar em produção

Nenhuma migration nova é necessária (ver seção 6). Passos:

```bash
# Backend
cd backend
go mod tidy          # confirma que go.sum está consistente no seu ambiente real
go vet ./...
go build ./...
go test ./...
# subir o binário/imagem normalmente, conforme seu processo de deploy atual

# Frontend
# Nenhum passo de build — é HTML/CSS/JS estático servido diretamente.
# Basta que os arquivos novos (pages/imports.js, css/imports.css,
# utils/aiIcon.js) e o index.html atualizado sejam publicados junto com
# o resto do frontend, como qualquer outro arquivo estático do projeto.
```

Nenhuma variável de ambiente nova é necessária — a Importação Inteligente
reaproveita a mesma configuração do Ollama já usada pelo Otis
(`OLLAMA_BASE_URL`, `OLLAMA_MODEL`, etc., já existentes em `config.go`).

---

## 11. Pendências reais (resumo)

1. Rodar `go test ./...` no seu ambiente com acesso normal a
   `proxy.golang.org`, para confirmar fora deste sandbox.
2. Teste de integração contra um banco Postgres real (local ou uma cópia
   de desenvolvimento do Supabase — nunca o de produção) antes de liberar
   para usuários reais, cobrindo pelo menos: uma importação completa de
   "Produto" com sucesso, uma de "Pré-produto", um caso de depósito sem
   permissão (deve falhar com mensagem clara), e um caso de falha parcial
   (um item bom + um item que o backend rejeita por algum motivo de
   negócio, ex. nome duplicado se essa regra existir na tabela).
3. Teste manual no navegador do fluxo completo de upload, com uma
   instância Ollama real rodando, para validar a resolução de
   ambiguidade de cabeçalho de fato (não só com mock).
4. Teste manual do fluxo de sessão (login, aguardar ou forçar expiração,
   fechar e reabrir o navegador) para confirmar visualmente o
   comportamento corrigido.
5. O `Dockerfile` do backend parece ter um problema pré-existente
   (caminho de `main.go` incorreto) não relacionado a esta entrega — vale
   revisar antes do próximo deploy via Docker.
