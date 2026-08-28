/**
 * imports.js — Importação Inteligente.
 *
 * Fluxo: seleção de planilha -> POST /imports/preview (nada é gravado)
 * -> revisão do usuário (editar/remover itens, ver duplicidades e
 * campos ausentes) -> POST /imports/commit (grava via os Services
 * existentes, item por item) -> resultado.
 *
 * Estado da tela inteira vive em variáveis de módulo simples (mesmo
 * padrão de outras páginas do sistema, ver inventory.js) porque cada
 * chamada a ImportsPage() já é uma remontagem completa da página pelo
 * router (ver router.js) — não precisa sobreviver a navegação.
 */
import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { notify } from "../components/notifications.js";
import { openModal } from "../components/modal.js";
import { guardedClick } from "../utils/security.js";
import { applyDateMask, isValidBRDate, randomFunnyDateError } from "../utils/validators.js";
import { starIcon } from "../utils/aiIcon.js";

const ACCEPTED_EXTENSIONS = [".xlsx", ".csv"];
const MAX_UPLOAD_BYTES = 10 * 1024 * 1024; // espelha importing.MaxUploadBytes no backend

// Rótulos e ícones para cada código de problema que o backend pode
// sinalizar (ver importing.FieldIssueCode) — mantidos aqui, não
// inventados: o texto exibido é sempre o `message` que o backend já
// gerou; isto só decide o ícone/cor.
const ISSUE_STYLE = {
  missing: { icon: "alert-triangle", tone: "warning" },
  invalid_format: { icon: "alert-triangle", tone: "danger" },
  invalid_quantity: { icon: "alert-triangle", tone: "danger" },
  expired: { icon: "clock-alert", tone: "danger" },
  expiring_soon: { icon: "clock", tone: "warning" },
  possible_duplicate: { icon: "copy", tone: "info" },
};

export function ImportsPage(root, ctx) {
  // Estado da sessão de importação atual — reiniciado a cada montagem
  // da página (nova navegação para /imports = começar do zero).
  const state = {
    stage: "idle", // idle | analyzing | preview | committing | done
    file: null,
    target: "inventory_item", // "inventory_item" | "pre_product"
    preview: null, // resposta de API.importPreview
    items: [], // cópia editável dos itens da prévia (com _removed)
    deposits: [],
    depositId: "",
    commitResult: null,
    filterOnlyProblems: false,
  };

  AppShell(root, ctx.path, (content) => {
    const container = el("div", { class: "imports-page" });
    content.append(pageHead(), container);
    renderIcons(content);
    renderStage(container, state);
  });
}

function pageHead() {
  return el("div", { class: "page-head" }, [
    el("div", {}, [
      el("h1", { class: "imports-title" }, [
        "Importação Inteligente ",
        el("span", { class: "imports-title-star" }, [starIcon(20, "otis-star")]),
      ]),
      el("p", { class: "muted", text: "Importe uma planilha de produtos e deixe o ANT-Stock analisar automaticamente os dados antes do cadastro." }),
    ]),
  ]);
}

/** Router central de estágio: decide o que renderizar dentro do container. */
function renderStage(container, state) {
  container.innerHTML = "";
  switch (state.stage) {
    case "idle":
      container.appendChild(idleStage(container, state));
      break;
    case "analyzing":
      container.appendChild(analyzingStage());
      break;
    case "preview":
      container.appendChild(previewStage(container, state));
      break;
    case "committing":
      container.appendChild(committingStage(state));
      break;
    case "done":
      container.appendChild(doneStage(container, state));
      break;
  }
  renderIcons(container);
}

// ── Estágio 1: seleção de arquivo ────────────────────────────────────

function idleStage(container, state) {
  const fileInput = el("input", {
    type: "file",
    accept: ".xlsx,.csv",
    style: "display:none",
    onchange: (e) => handleFileSelected(container, state, e.target.files?.[0]),
  });

  const dropzone = el("div", { class: "import-dropzone" }, [
    el("div", { class: "import-dropzone-icon" }, [el("i", { "data-lucide": "file-spreadsheet" })]),
    el("p", { class: "import-dropzone-title", text: "Selecionar planilha" }),
    el("p", { class: "muted", text: "ou arraste sua planilha aqui" }),
    el("button", { type: "button", class: "btn btn-primary", onclick: guardedClick(() => fileInput.click()) }, [
      el("i", { "data-lucide": "upload" }), "Selecionar planilha",
    ]),
    fileInput,
  ]);

  dropzone.addEventListener("dragover", (e) => { e.preventDefault(); dropzone.classList.add("dragover"); });
  dropzone.addEventListener("dragleave", () => dropzone.classList.remove("dragover"));
  dropzone.addEventListener("drop", (e) => {
    e.preventDefault();
    dropzone.classList.remove("dragover");
    const file = e.dataTransfer?.files?.[0];
    if (file) handleFileSelected(container, state, file);
  });

  const info = el("div", { class: "import-info-row" }, [
    el("div", { class: "import-info-item" }, [
      el("i", { "data-lucide": "check-circle-2" }),
      el("span", {}, ["Formatos aceitos: ", el("strong", { text: "XLSX • CSV" })]),
    ]),
    el("div", { class: "import-info-item" }, [
      el("i", { "data-lucide": "sparkles" }),
      el("span", { text: "A IA analisará os dados antes da importação." }),
    ]),
    el("div", { class: "import-info-item" }, [
      el("i", { "data-lucide": "shield-check" }),
      el("span", { text: "Você poderá revisar tudo antes de salvar." }),
    ]),
  ]);

  return el("div", { class: "card card-pad import-card" }, [dropzone, info]);
}

/** Validação client-side de conveniência (feedback imediato) — a validação que vale é sempre a do backend. */
function validateFileClientSide(file) {
  const lowerName = file.name.toLowerCase();
  const hasAcceptedExtension = ACCEPTED_EXTENSIONS.some((ext) => lowerName.endsWith(ext));
  if (!hasAcceptedExtension) {
    return "Formato não suportado. Envie um arquivo .xlsx ou .csv.";
  }
  if (file.size === 0) {
    return "O arquivo está vazio.";
  }
  if (file.size > MAX_UPLOAD_BYTES) {
    return "Arquivo excede o tamanho máximo permitido (10 MB).";
  }
  return null;
}

async function handleFileSelected(container, state, file) {
  if (!file) return;
  const clientError = validateFileClientSide(file);
  if (clientError) {
    notify(clientError, "error");
    return;
  }

  state.file = file;
  state.stage = "analyzing";
  renderStage(container, state);

  try {
    const preview = await API.importPreview(file, state.target);
    state.preview = preview;
    state.items = (preview.items || []).map((item, idx) => ({
      ...item,
      _key: `${item.row_index}-${idx}`,
      _removed: false,
    }));
    await ensureDepositsLoaded(state);
    state.stage = "preview";
    renderStage(container, state);
  } catch (err) {
    state.stage = "idle";
    renderStage(container, state);
    notify(err.message || "Não foi possível analisar a planilha.", "error");
  }
}

async function ensureDepositsLoaded(state) {
  try {
    state.deposits = await API.deposits({ scope: "stock", classId: session.classId });
    if (!state.depositId && state.deposits.length === 1) {
      state.depositId = state.deposits[0].id; // único depósito disponível: pré-seleciona
    }
  } catch {
    state.deposits = [];
  }
}

// ── Estágio 2: analisando ────────────────────────────────────────────

function analyzingStage() {
  const steps = ["Lendo produtos...", "Identificando colunas...", "Validando dados...", "Preparando importação..."];
  return el("div", { class: "card card-pad import-card import-processing" }, [
    el("div", { class: "import-spinner" }, [el("span", { class: "spinner-ring" })]),
    el("p", { class: "import-processing-title", text: "Analisando planilha..." }),
    el("ul", { class: "import-processing-steps stagger" }, steps.map((s) => el("li", { text: s }))),
  ]);
}

// ── Estágio 3: prévia ────────────────────────────────────────────────

function previewStage(container, state) {
  const wrap = el("div", { class: "import-preview" });

  wrap.appendChild(previewSummaryCard(state));
  wrap.appendChild(targetAndDepositCard(container, state));

  const visibleItems = state.items.filter((it) => !it._removed && (!state.filterOnlyProblems || !it.ready));

  const listHead = el("div", { class: "import-list-head" }, [
    el("h3", { text: `${visibleItems.length} de ${activeItemCount(state)} itens` }),
    el("label", { class: "import-filter-toggle" }, [
      el("input", {
        type: "checkbox",
        checked: state.filterOnlyProblems,
        onchange: (e) => { state.filterOnlyProblems = e.target.checked; renderStage(container, state); },
      }),
      el("span", { text: "Mostrar somente problemas" }),
    ]),
  ]);
  wrap.appendChild(listHead);

  if (state.preview.unmapped_headers?.length) {
    wrap.appendChild(el("div", { class: "import-unmapped-note" }, [
      el("i", { "data-lucide": "info" }),
      el("span", {}, [
        "Colunas não reconhecidas e ignoradas: ",
        el("strong", { text: state.preview.unmapped_headers.join(", ") }),
      ]),
    ]));
  }

  const grid = el("div", { class: "import-item-grid" });
  if (visibleItems.length === 0) {
    grid.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, [
      state.filterOnlyProblems ? "Nenhum item com problemas — tudo pronto." : "Nenhum item para exibir.",
    ]));
  } else {
    for (const item of visibleItems) {
      grid.appendChild(importItemCard(container, state, item));
    }
  }
  wrap.appendChild(grid);

  wrap.appendChild(previewActions(container, state));

  return wrap;
}

function activeItemCount(state) {
  return state.items.filter((it) => !it._removed).length;
}

function previewSummaryCard(state) {
  const active = state.items.filter((it) => !it._removed);
  const totalUnits = active.reduce((sum, it) => sum + (Number(it.quantity) || 0), 0);
  const readyCount = active.filter((it) => it.ready).length;
  const attentionCount = active.length - readyCount;

  const aiNote = state.preview.ai_used
    ? el("div", { class: "import-ai-note" }, [starIcon(14, "otis-star"), el("span", { text: "A IA ajudou a identificar algumas colunas desta planilha." })])
    : null;

  return el("div", { class: "card card-pad import-summary-card" }, [
    el("div", { class: "import-summary-head" }, [
      el("div", {}, [
        el("h3", { text: "Importação encontrada" }),
        el("p", { class: "muted", text: `${active.length} itens identificados · ${totalUnits} unidades` }),
      ]),
    ]),
    el("div", { class: "import-summary-chips" }, [
      el("span", { class: "chip chip-success", text: `${readyCount} prontos` }),
      attentionCount > 0 ? el("span", { class: "chip chip-warning", text: `${attentionCount} precisam de atenção` }) : null,
    ].filter(Boolean)),
    aiNote,
  ].filter(Boolean));
}

function targetAndDepositCard(container, state) {
  const targetRadios = el("div", { class: "import-target-options" }, [
    targetRadio(container, state, "inventory_item", "Produto", "Cadastra no estoque do depósito selecionado, com quantidade, lote e validade."),
    targetRadio(container, state, "pre_product", "Pré-produto", "Cadastra no catálogo permanente (sem quantidade/lote/validade)."),
  ]);

  const depositField = state.target === "inventory_item" ? depositSelectField(state) : null;

  return el("div", { class: "card card-pad import-target-card" }, [
    el("h3", { text: "Importar como" }),
    targetRadios,
    depositField,
  ].filter(Boolean));
}

function targetRadio(container, state, value, label, description) {
  const id = `import-target-${value}`;
  return el("label", { class: "import-target-option" + (state.target === value ? " active" : ""), for: id }, [
    el("input", {
      type: "radio", name: "import-target", id, value, checked: state.target === value,
      onchange: async () => {
        state.target = value;
        // Reprocessa a prévia com o novo destino: campos obrigatórios
        // mudam (ver RequiredFieldsFor no backend), então re-chamamos o
        // preview com o mesmo arquivo em vez de tentar reclassificar no
        // frontend — mantém uma única fonte de verdade das regras.
        state.stage = "analyzing";
        renderStage(container, state);
        try {
          const preview = await API.importPreview(state.file, state.target);
          state.preview = preview;
          state.items = (preview.items || []).map((item, idx) => ({ ...item, _key: `${item.row_index}-${idx}`, _removed: false }));
        } catch (err) {
          notify(err.message || "Não foi possível reprocessar a planilha.", "error");
        }
        state.stage = "preview";
        renderStage(container, state);
      },
    }),
    el("div", {}, [
      el("div", { class: "import-target-option-label", text: label }),
      el("div", { class: "import-target-option-desc muted", text: description }),
    ]),
  ]);
}

function depositSelectField(state) {
  if (!state.deposits.length) {
    return el("div", { class: "field" }, [
      el("label", { text: "Depósito" }),
      el("div", { class: "muted", text: "Nenhum depósito de estoque disponível para este usuário." }),
    ]);
  }
  const select = el("select", {
    class: "select",
    onchange: (e) => { state.depositId = e.target.value; },
  }, [
    el("option", { value: "", text: "Selecione o depósito" }),
    ...state.deposits.map((d) => el("option", { value: d.id, text: d.name, selected: state.depositId === d.id })),
  ]);
  select.value = state.depositId || "";
  return el("div", { class: "field" }, [el("label", { text: "Depósito" }), select]);
}

function importItemCard(container, state, item) {
  const blockingCodes = ["missing", "invalid_format", "invalid_quantity"];
  const blockingIssues = (item.issues || []).filter((i) => blockingCodes.includes(i.code));
  const infoIssues = (item.issues || []).filter((i) => !blockingCodes.includes(i.code));

  const statusChip = item.ready
    ? el("span", { class: "chip chip-success import-status-chip" }, [el("i", { "data-lucide": "check" }), "Pronto para importar"])
    : el("span", { class: "chip chip-warning import-status-chip" }, [el("i", { "data-lucide": "alert-triangle" }), "Precisa de atenção"]);

  const rows = [
    ["Quantidade", item.quantity > 0 ? String(item.quantity) : "AUSENTE"],
    ["Lote", item.lot_number || (state.target === "inventory_item" ? "AUSENTE" : null)],
    ["Validade", item.expiry_date || (state.target === "inventory_item" ? "AUSENTE" : null)],
  ].filter(([, value]) => value !== null);
  if (state.target === "pre_product") {
    rows.push(["Unidade", item.unit || "AUSENTE"]);
  }

  const detailRows = rows.map(([label, value]) => {
    const missing = value === "AUSENTE";
    return el("div", { class: "import-item-row" }, [
      el("span", { class: "import-item-row-label", text: label }),
      el("span", { class: "import-item-row-value" + (missing ? " missing" : ""), text: value }),
    ]);
  });

  const issueNodes = [...blockingIssues, ...infoIssues].map((issue) => {
    const style = ISSUE_STYLE[issue.code] || { icon: "info", tone: "info" };
    return el("div", { class: `import-issue import-issue-${style.tone}` }, [
      el("i", { "data-lucide": style.icon }),
      el("span", { text: issue.message }),
    ]);
  });

  const actions = el("div", { class: "import-item-actions" }, [
    el("button", { type: "button", class: "btn btn-outline btn-sm", onclick: guardedClick(() => openEditItemModal(container, state, item)) }, [
      el("i", { "data-lucide": "pencil" }), "Editar",
    ]),
    el("button", { type: "button", class: "btn btn-ghost btn-sm", onclick: guardedClick(() => removeItem(container, state, item)) }, [
      el("i", { "data-lucide": "trash-2" }), "Remover",
    ]),
  ]);

  return el("div", { class: "card import-item-card" }, [
    el("div", { class: "import-item-head" }, [
      el("h4", { text: item.name || "(sem nome)" }),
      statusChip,
    ]),
    el("div", { class: "import-item-details" }, detailRows),
    issueNodes.length ? el("div", { class: "import-item-issues" }, issueNodes) : null,
    actions,
  ].filter(Boolean));
}

function removeItem(container, state, item) {
  item._removed = true;
  renderStage(container, state);
  notify(`"${item.name || "Item"}" removido da importação.`, "info");
}

// ── Edição individual de item ────────────────────────────────────────

function openEditItemModal(container, state, item) {
  const isInventory = state.target === "inventory_item";

  const nameInput = el("input", { class: "input", type: "text", value: item.name || "" });
  const nameError = el("div", { class: "error-text" });
  const qtyInput = el("input", { class: "input", type: "number", min: "0", value: item.quantity || "" });
  const lotInput = el("input", { class: "input", type: "text", value: item.lot_number || "" });
  const dateInput = el("input", { class: "input", type: "text", placeholder: "DD/MM/AAAA", maxlength: "10", value: item.expiry_date || "" });
  const dateError = el("div", { class: "error-text" });
  dateInput.addEventListener("input", () => { dateInput.value = applyDateMask(dateInput.value); });
  const unitInput = el("input", { class: "input", type: "text", value: item.unit || "", placeholder: "ex.: unidade, caixa, kg" });
  const brandInput = el("input", { class: "input", type: "text", value: item.brand || "" });
  const notesInput = el("textarea", { class: "textarea", text: item.notes || "" });

  const fields = [
    el("div", { class: "field" }, [el("label", { text: "Produto" }), nameInput, nameError]),
  ];
  if (isInventory) {
    fields.push(
      el("div", { class: "field" }, [el("label", { text: "Quantidade" }), qtyInput]),
      el("div", { class: "field" }, [el("label", { text: "Lote" }), lotInput]),
      el("div", { class: "field" }, [el("label", { text: "Validade" }), dateInput, dateError]),
    );
  } else {
    fields.push(el("div", { class: "field" }, [el("label", { text: "Unidade de medida" }), unitInput]));
  }
  fields.push(
    el("div", { class: "field" }, [el("label", { text: "Marca" }), brandInput]),
    el("div", { class: "field" }, [el("label", { text: "Observações" }), notesInput]),
  );

  const body = el("div", { class: "import-edit-form" }, fields);

  // openModal() genérico (ver components/modal.js) sempre fecha ao
  // confirmar, não há como "recusar" o fechamento a partir de
  // onConfirm — por isso a validação acontece ANTES de abrir o modal
  // de confirmação: os campos são validados a cada alteração
  // (mostrando erro inline, mesmo padrão de error-text usado em
  // inventoryModal.js), e o botão de salvar do modal, se os dados
  // ainda estiverem inválidos no momento do clique, apenas notifica e
  // não aplica a edição — o usuário reabre "Editar" para tentar de
  // novo, já com o erro visível nos campos.
  function currentlyValid() {
    let valid = true;
    if (!nameInput.value.trim()) {
      nameError.textContent = "Nome é obrigatório.";
      valid = false;
    } else {
      nameError.textContent = "";
    }
    if (isInventory && dateInput.value && !isValidBRDate(dateInput.value)) {
      dateError.textContent = randomFunnyDateError();
      valid = false;
    } else {
      dateError.textContent = "";
    }
    return valid;
  }
  nameInput.addEventListener("input", currentlyValid);
  dateInput.addEventListener("input", () => { dateInput.value = applyDateMask(dateInput.value); currentlyValid(); });

  openModal({
    title: `Editar item — linha ${item.row_index}`,
    body,
    primaryLabel: "Salvar",
    onConfirm: () => {
      if (!currentlyValid()) {
        notify("Corrija os campos destacados antes de salvar.", "error");
        return;
      }

      item.name = nameInput.value.trim();
      item.brand = brandInput.value.trim();
      item.notes = notesInput.value.trim();
      if (isInventory) {
        item.quantity = Number(qtyInput.value) || 0;
        item.lot_number = lotInput.value.trim();
        item.expiry_date = dateInput.value.trim();
      } else {
        item.unit = unitInput.value.trim();
      }

      // Reavalia localmente se o item ficou "pronto" após a edição —
      // espelha as mesmas regras de RequiredFieldsFor do backend, só
      // para feedback imediato; o backend revalida tudo de novo no
      // commit (nunca confia nesta decisão do frontend).
      item.issues = recomputeIssuesClientSide(item, isInventory);
      item.ready = !item.issues.some((i) => ["missing", "invalid_format", "invalid_quantity"].includes(i.code));

      renderStage(container, state);
      notify("Item atualizado.", "success");
    },
  });
}

/** Réplica simplificada, só para feedback de UX, das regras de campo obrigatório do backend (ver importing.applyRequiredFieldIssues). */
function recomputeIssuesClientSide(item, isInventory) {
  const issues = (item.issues || []).filter((i) => i.code === "possible_duplicate"); // preserva avisos de duplicidade já calculados no preview
  if (!item.name) issues.push({ field: "name", code: "missing", message: "Nome do produto não informado" });
  if (isInventory) {
    if (!item.quantity || item.quantity <= 0) issues.push({ field: "quantity", code: "missing", message: "Quantidade não informada" });
    if (!item.lot_number) issues.push({ field: "lot_number", code: "missing", message: "Lote não informado" });
    if (!item.expiry_date) issues.push({ field: "expiry_date", code: "missing", message: "Validade não informada" });
    else if (!isValidBRDate(item.expiry_date)) issues.push({ field: "expiry_date", code: "invalid_format", message: "Data de validade inválida: " + item.expiry_date });
  } else if (!item.unit) {
    issues.push({ field: "unit", code: "missing", message: "Unidade de medida não informada" });
  }
  return issues;
}

// ── Ações da prévia (cancelar / remover todos / importar) ───────────

function previewActions(container, state) {
  const active = state.items.filter((it) => !it._removed);
  const readyCount = active.filter((it) => it.ready).length;
  const attentionCount = active.length - readyCount;

  return el("div", { class: "import-preview-actions" }, [
    el("button", { type: "button", class: "btn btn-ghost", onclick: guardedClick(() => resetToIdle(container, state)) }, ["Cancelar"]),
    el("button", { type: "button", class: "btn btn-outline", onclick: guardedClick(() => confirmRemoveAll(container, state)) }, ["Remover todos"]),
    el("button", {
      type: "button",
      class: "btn btn-primary",
      disabled: active.length === 0,
      onclick: guardedClick(() => openConfirmSummary(container, state, readyCount, attentionCount)),
    }, [el("i", { "data-lucide": "check-circle" }), "Importar produtos"]),
  ]);
}

function resetToIdle(container, state) {
  state.stage = "idle";
  state.file = null;
  state.preview = null;
  state.items = [];
  renderStage(container, state);
}

function confirmRemoveAll(container, state) {
  openModal({
    title: "Remover todos os itens?",
    body: "Todos os itens desta importação serão removidos da lista. Esta ação não pode ser desfeita.",
    primaryLabel: "Remover todos",
    danger: true,
    onConfirm: () => {
      for (const item of state.items) item._removed = true;
      renderStage(container, state);
      notify("Todos os itens foram removidos.", "info");
    },
  });
}

function openConfirmSummary(container, state, readyCount, attentionCount) {
  if (state.target === "inventory_item" && !state.depositId) {
    notify("Selecione um depósito antes de importar.", "error");
    return;
  }

  const active = state.items.filter((it) => !it._removed);
  const totalUnits = active.reduce((sum, it) => sum + (Number(it.quantity) || 0), 0);

  const body = el("div", { class: "import-confirm-summary" }, [
    el("p", {}, [el("strong", { text: String(active.length) }), " produtos"]),
    state.target === "inventory_item" ? el("p", {}, [el("strong", { text: String(totalUnits) }), " unidades"]) : null,
    el("p", { class: "chip chip-success", text: `${readyCount} prontos` }),
    attentionCount > 0 ? el("p", { class: "chip chip-warning", text: `${attentionCount} precisam de atenção — serão ignorados nesta importação` }) : null,
  ].filter(Boolean));

  openModal({
    title: "Resumo da importação",
    body,
    primaryLabel: attentionCount > 0 ? `Importar ${readyCount} produtos válidos` : "Importar produtos",
    onConfirm: () => runCommit(container, state),
  });
}

// ── Estágio 4: importando (progresso) ────────────────────────────────

function committingStage(state) {
  const total = state._commitTotal || 0;
  const done = state._commitDone || 0;
  const pct = total > 0 ? Math.round((done / total) * 100) : 0;
  return el("div", { class: "card card-pad import-card import-processing" }, [
    el("div", { class: "import-spinner" }, [el("span", { class: "spinner-ring" })]),
    el("p", { class: "import-processing-title", text: "Importando produtos..." }),
    el("div", { class: "import-progress-bar" }, [
      el("div", { class: "import-progress-fill", style: `width:${pct}%` }),
    ]),
    el("p", { class: "muted", text: `${done} / ${total}` }),
  ]);
}

async function runCommit(container, state) {
  const active = state.items.filter((it) => !it._removed && it.ready);
  state.stage = "committing";
  state._commitTotal = active.length;
  state._commitDone = 0;
  renderStage(container, state);

  // O backend processa a lista inteira numa única chamada (cada item é
  // uma operação independente lá dentro — ver ImportService.Commit), o
  // que já é rápido o bastante para o volume tratado por esta
  // funcionalidade (até importing.MaxDataRows linhas). A barra de
  // progresso é preenchida ao concluir a chamada — não há como refletir
  // progresso item-a-item sem um endpoint de streaming, fora do escopo
  // desta V1.
  try {
    const result = await API.importCommit({
      depositId: state.depositId,
      target: state.target,
      items: active.map(toCommitItem),
    });
    state._commitDone = state._commitTotal;
    state.commitResult = result;
    state.stage = "done";
    renderStage(container, state);
  } catch (err) {
    state.stage = "preview";
    renderStage(container, state);
    notify(err.message || "Não foi possível concluir a importação.", "error");
  }
}

function toCommitItem(item) {
  return {
    row_index: item.row_index,
    name: item.name,
    quantity: Number(item.quantity) || 0,
    lot_number: item.lot_number || "",
    expiry_date: item.expiry_date || "",
    sku: item.sku || "",
    brand: item.brand || "",
    min_quantity: Number(item.min_quantity) || 0,
    category_id: item.category_id || null,
    notes: item.notes || "",
    unit: item.unit || "",
  };
}

// ── Estágio 5: resultado ─────────────────────────────────────────────

function doneStage(container, state) {
  const result = state.commitResult;
  const hasFailures = result.total_failed > 0;
  const failedResults = (result.results || []).filter((r) => !r.success);

  const summary = el("div", { class: "card card-pad import-done-card" }, [
    el("div", { class: "import-done-icon" + (hasFailures ? " has-warning" : "") }, [
      el("i", { "data-lucide": hasFailures ? "alert-triangle" : "check-circle-2" }),
    ]),
    el("h2", { text: "Importação concluída" }),
    el("div", { class: "import-done-stats" }, [
      el("div", { class: "import-done-stat" }, [el("strong", { text: String(result.total_analyzed) }), el("span", { text: "analisados" })]),
      el("div", { class: "import-done-stat" }, [el("strong", { text: String(result.total_imported), class: "text-success" }), el("span", { text: "importados" })]),
      result.total_failed > 0 ? el("div", { class: "import-done-stat" }, [el("strong", { text: String(result.total_failed), class: "text-danger" }), el("span", { text: "não importados" })]) : null,
    ].filter(Boolean)),
  ]);

  const failuresBlock = hasFailures ? el("div", { class: "card card-pad import-failures-card" }, [
    el("h3", { text: `${failedResults.length} produto(s) não puderam ser importados` }),
    el("div", { class: "import-failures-list" }, failedResults.map((r) =>
      el("div", { class: "import-failure-row" }, [
        el("span", { class: "import-failure-name", text: r.name || `Linha ${r.row_index}` }),
        el("span", { class: "import-failure-reason muted", text: r.error }),
      ]),
    )),
  ]) : null;

  const actions = el("div", { class: "import-done-actions" }, [
    el("button", { type: "button", class: "btn btn-outline", onclick: guardedClick(() => resetToIdle(container, state)) }, [
      el("i", { "data-lucide": "upload" }), "Nova importação",
    ]),
    el("a", { class: "btn btn-primary", href: "#/inventory" }, [
      el("i", { "data-lucide": "package" }), "Ver produtos",
    ]),
  ]);

  return el("div", { class: "import-done-wrap" }, [summary, failuresBlock, actions].filter(Boolean));
}
