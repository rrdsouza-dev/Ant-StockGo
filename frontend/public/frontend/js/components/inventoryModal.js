/**
 * inventoryModal.js — Modais de item de estoque e de movimentação
 * (entrada/saída). Substitui o antigo productModal.js: não existe mais
 * "produto isolado" — todo modal aqui opera sobre um item de inventário
 * pertencente a um depósito.
 */
import { el, renderIcons } from "../utils/helpers.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { notify } from "./notifications.js";
import { openModal } from "./modal.js";
import { applyDateMask, isValidBRDate, randomFunnyDateError } from "../utils/validators.js";
import { guardedClick } from "../utils/security.js";

/** Converte "2026-12-31" (ou ISO completo) para "31/12/2026" para exibição. */
function isoToBRDate(iso) {
  if (!iso) return "";
  const datePart = String(iso).slice(0, 10); // "AAAA-MM-DD"
  const [y, m, d] = datePart.split("-");
  if (!y || !m || !d) return "";
  return `${d}/${m}/${y}`;
}

/** Monta um <select> numérico de 1 a `max` (usado para corredor/torre/prateleira). */
function numberSelect(max, selected) {
  const select = el("select", { class: "select" }, [
    el("option", { value: "", text: "—" }),
  ]);
  for (let i = 1; i <= max; i++) {
    select.appendChild(el("option", { value: String(i), text: String(i), selected: selected === i }));
  }
  if (selected) select.value = String(selected);
  return select;
}

/** Monta o <select> de posição: A1 até A10. */
function positionSelect(selected) {
  const select = el("select", { class: "select" }, [
    el("option", { value: "", text: "—" }),
  ]);
  for (let i = 1; i <= 10; i++) {
    const value = `A${i}`;
    select.appendChild(el("option", { value, text: value }));
  }
  if (selected) select.value = selected;
  return select;
}

/**
 * openInventoryItemModal — cria ou edita um item de estoque, com todos os
 * campos do cadastro estendido (validade, lote, categoria, localização,
 * observações). Fluxo: chama API.createInventoryItem / API.updateInventoryItem.
 * Restrito à gestão na UI (o backend também recusaria para professor).
 */
export async function openInventoryItemModal({ depositId, item, onSave }) {
  const isEdit = !!item;
  const loc = item?.location || {};

  let categories = [];
  try {
    categories = await API.categories();
  } catch {
    // Segue sem categorias pré-carregadas; o select fica só com "Nenhuma".
  }

  const f = {
    name: el("input", { class: "input", value: item?.name || "", placeholder: "Nome do item *" }),
    sku: el("input", { class: "input", value: item?.sku || "", placeholder: "Código / SKU (opcional)" }),
    brand: el("input", { class: "input", value: item?.brand || "", placeholder: "Marca (opcional)" }),
    min: el("input", { class: "input", type: "number", min: "0", value: item?.min_quantity ?? 0, placeholder: "Quantidade mínima" }),
    expiry: el("input", {
      class: "input", value: isoToBRDate(item?.expiry_date), placeholder: "DD/MM/AAAA", inputmode: "numeric", maxlength: "10",
    }),
    // Lote é obrigatório ao entrar no estoque — mesmo princípio de expiry_date,
    // validado tanto aqui (feedback imediato) quanto no backend (fonte da verdade).
    lot: el("input", { class: "input", value: item?.lot_number || "", placeholder: "Número do lote *" }),
    notes: el("textarea", { class: "input", rows: "3", placeholder: "Observações (opcional)", text: item?.notes || "" }),
  };
  f.expiry.addEventListener("input", () => { f.expiry.value = applyDateMask(f.expiry.value); expiryErr.textContent = ""; });

  // ── Categoria + botão "+" ──────────────────────────────────
  // Criação de categoria disponível para qualquer usuário autenticado
  // (gestão e professor) — mesmo endpoint, mesmo modal, sem duplicar
  // lógica (ver routes.go: POST /categories não é mais gestaoOnly).
  const categorySelect = el("select", { class: "select" }, [
    el("option", { value: "", text: "Nenhuma" }),
    ...categories.map((c) => el("option", { value: c.id, text: c.name, selected: item?.category_id === c.id })),
  ]);
  if (item?.category_id) categorySelect.value = item.category_id;
  const addCategoryBtn = el("button", { type: "button", class: "icon-btn", title: "Nova categoria" }, [el("i", { "data-lucide": "plus" })]);
  addCategoryBtn.addEventListener("click", () => openCategoryModal((created) => {
    categorySelect.appendChild(el("option", { value: created.id, text: created.name }));
    categorySelect.value = created.id;
  }));
  const categoryRow = el("div", { class: "field-inline-add" }, [categorySelect, addCategoryBtn]);

  // ── Localização genérica: corredor / torre / prateleira / posição ──
  const aisleSelect = numberSelect(10, loc.aisle);
  const towerSelect = numberSelect(10, loc.tower);
  const shelfSelect = numberSelect(10, loc.shelf);
  const positionSel = positionSelect(loc.position);

  const expiryErr = el("div", { class: "error-text" });
  const lotErr = el("div", { class: "error-text" });
  const errEl = el("div", { class: "error-text" });
  const saveBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "save" }), isEdit ? "Salvar" : "Criar"]);
  const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });

  const card = el("div", { class: "modal modal-lg" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: isEdit ? "Editar item de estoque" : "Novo item de estoque" })]),
    el("div", { class: "product-modal-body" }, [
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Nome *" }), f.name]),
      el("div", { class: "form-grid-2" }, [
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Código / SKU" }), f.sku]),
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Marca" }), f.brand]),
      ]),
      el("div", { class: "form-grid-2" }, [
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Data de validade *" }), f.expiry, expiryErr]),
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Número do lote *" }), f.lot, lotErr]),
      ]),
      el("div", { class: "form-grid-2" }, [
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Quantidade mínima" }), f.min]),
        el("div", {}),
      ]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Categoria" }), categoryRow]),
      el("div", { class: "field" }, [
        el("label", { class: "field-label", text: "Localização" }),
        el("div", { class: "form-grid-4" }, [
          el("div", {}, [el("label", { class: "field-sublabel", text: "Corredor" }), aisleSelect]),
          el("div", {}, [el("label", { class: "field-sublabel", text: "Torre" }), towerSelect]),
          el("div", {}, [el("label", { class: "field-sublabel", text: "Prateleira" }), shelfSelect]),
          el("div", {}, [el("label", { class: "field-sublabel", text: "Posição" }), positionSel]),
        ]),
      ]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Observações" }), f.notes]),
      errEl,
    ]),
    el("div", { class: "modal-actions" }, [cancelBtn, saveBtn]),
  ]);

  const backdrop = el("div", { class: "modal-backdrop" }, [card]);
  const close = () => backdrop.remove();
  backdrop.addEventListener("click", (e) => { if (e.target === backdrop) close(); });
  cancelBtn.addEventListener("click", close);

  saveBtn.addEventListener("click", async () => {
    const name = f.name.value.trim();
    errEl.textContent = "";
    expiryErr.textContent = "";
    lotErr.textContent = "";
    if (!name) { errEl.textContent = "Nome é obrigatório."; return; }

    const expiryValue = f.expiry.value.trim();
    if (!expiryValue || !isValidBRDate(expiryValue)) {
      expiryErr.textContent = randomFunnyDateError();
      return;
    }

    // Lote obrigatório na entrada de estoque — mesma regra aplicada no
    // backend (InventoryService.Create/Update); aqui é só feedback rápido.
    const lotValue = f.lot.value.trim();
    if (!lotValue) {
      lotErr.textContent = "Número do lote é obrigatório.";
      return;
    }

    saveBtn.disabled = true;
    try {
      const data = {
        name,
        sku: f.sku.value.trim(),
        brand: f.brand.value.trim(),
        minQuantity: Number(f.min.value) || 0,
        expiryDate: expiryValue,
        lotNumber: lotValue,
        categoryId: categorySelect.value || null,
        notes: f.notes.value.trim(),
        location: {
          aisle: Number(aisleSelect.value) || 0,
          tower: Number(towerSelect.value) || 0,
          shelf: Number(shelfSelect.value) || 0,
          position: positionSel.value || "",
        },
      };
      if (isEdit) await API.updateInventoryItem(item.id, data);
      else await API.createInventoryItem({ depositId, ...data });
      notify(isEdit ? "Item atualizado!" : "Item criado!", "success");
      close();
      onSave?.();
    } catch (err) {
      errEl.textContent = err.message || "Erro ao salvar item.";
    } finally {
      saveBtn.disabled = false;
    }
  });

  document.body.appendChild(backdrop);
  renderIcons(backdrop);
  setTimeout(() => f.name.focus(), 80);
}

/**
 * openCategoryModal — pequeno modal para cadastrar uma nova categoria
 * (botão "+" ao lado do campo Categoria). Chama onCreated(category) para
 * que o formulário de item selecione a categoria recém-criada na hora,
 * sem precisar recarregar nada.
 */
/**
 * openCategoryModal — cria uma nova categoria e, para a gestão, também
 * permite excluir categorias já existentes (item 3 da especificação de
 * melhorias). Chamado tanto pelo botão "+" do formulário de item de
 * estoque quanto pelo do Pré-Produto — mesmo modal, mesma lista, sem
 * duplicar nada: como a categoria é uma entidade única e compartilhada
 * (não há uma tabela separada por depósito ou por turma), excluir aqui
 * já vale para qualquer tela que use categorias.
 */
async function openCategoryModal(onCreated) {
  const nameInput = el("input", { class: "input", placeholder: "Nome da categoria *" });
  const errEl = el("div", { class: "error-text" });
  const saveBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "save" }), "Criar"]);
  const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });
  const isGestao = session.user?.role === "gestao";

  const existingList = el("div", { class: "checkbox-list", style: "max-height:160px" }, [
    el("div", { class: "muted", style: "padding:14px;text-align:center", text: "Carregando…" }),
  ]);

  const card = el("div", { class: "modal modal-sm" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: "Categorias" })]),
    el("div", { class: "product-modal-body" }, [
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Nova categoria" }), nameInput]),
      errEl,
      isGestao ? el("div", { class: "field" }, [
        el("label", { class: "field-label", text: "Categorias existentes" }),
        existingList,
      ]) : el("span"),
    ]),
    el("div", { class: "modal-actions" }, [cancelBtn, saveBtn]),
  ]);
  const backdrop = el("div", { class: "modal-backdrop" }, [card]);
  const close = () => backdrop.remove();
  backdrop.addEventListener("click", (e) => { if (e.target === backdrop) close(); });
  cancelBtn.addEventListener("click", close);

  async function loadExisting() {
    if (!isGestao) return;
    try {
      const categories = await API.categories();
      existingList.innerHTML = "";
      if (!categories.length) {
        existingList.appendChild(el("div", { class: "muted", style: "padding:14px;text-align:center", text: "Nenhuma categoria cadastrada." }));
        return;
      }
      categories.forEach((cat) => {
        existingList.appendChild(el("div", { class: "checkbox-list-item", style: "justify-content:space-between;cursor:default" }, [
          el("div", { class: "checkbox-list-name", text: cat.name }),
          el("button", { class: "icon-btn", title: "Excluir categoria", onclick: () => confirmDeleteCategory(cat) }, [el("i", { "data-lucide": "trash-2" })]),
        ]));
      });
      renderIcons(existingList);
    } catch {
      existingList.innerHTML = "";
      existingList.appendChild(el("div", { class: "muted", style: "padding:14px;text-align:center", text: "Erro ao carregar categorias." }));
    }
  }

  function confirmDeleteCategory(cat) {
    openModal({
      title: "Excluir categoria",
      body: `Excluir a categoria "${cat.name}"? Itens que já usam essa categoria não são afetados — apenas deixam de ter categoria.`,
      primaryLabel: "Excluir",
      danger: true,
      onConfirm: async () => {
        await API.deleteCategory(cat.id);
        notify("Categoria excluída.", "warning");
        loadExisting();
      },
    });
  }

  saveBtn.addEventListener("click", async () => {
    const name = nameInput.value.trim();
    if (!name) { errEl.textContent = "Nome é obrigatório."; return; }
    saveBtn.disabled = true;
    try {
      const created = await API.createCategory(name);
      notify("Categoria criada!", "success");
      nameInput.value = "";
      onCreated?.(created);
      loadExisting();
    } catch (err) {
      errEl.textContent = err.message || "Erro ao criar categoria.";
    } finally {
      saveBtn.disabled = false;
    }
  });

  document.body.appendChild(backdrop);
  renderIcons(backdrop);
  setTimeout(() => nameInput.focus(), 80);
  loadExisting();
}

/**
 * openMoveModal — registra uma entrada ou saída de estoque para um item já
 * identificado. Disponível para professor (dentro dos seus depósitos) e
 * gestão. Fluxo: chama API.moveStock, que gera o StockMovement no backend.
 */
export function openMoveModal({ item, type = "in", onSave }) {
  let currentType = type;

  const typeToggle = el("div", { class: "perfil-selector" });
  const options = [
    { value: "in", label: "Entrada", icon: "arrow-down-circle" },
    { value: "out", label: "Saída", icon: "arrow-up-circle" },
  ];
  options.forEach(({ value, label }) => {
    const btn = el("button", {
      type: "button",
      class: "btn-perfil" + (value === currentType ? " active" : ""),
      text: label,
    });
    btn.addEventListener("click", () => {
      currentType = value;
      typeToggle.querySelectorAll(".btn-perfil").forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");
    });
    typeToggle.appendChild(btn);
  });

  const qtyInput = el("input", { class: "input", type: "number", min: "1", value: "1", placeholder: "Quantidade" });
  const noteInput = el("input", { class: "input", placeholder: "Observação (opcional)" });
  const errEl = el("div", { class: "error-text" });
  const saveBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "check" }), "Confirmar"]);
  const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });

  const card = el("div", { class: "modal" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: `Movimentar: ${item.name}` })]),
    el("div", { class: "product-modal-body" }, [
      el("p", { class: "muted", style: "margin-bottom:12px", text: `Saldo atual: ${item.quantity} un.` }),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Tipo" }), typeToggle]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Quantidade *" }), qtyInput]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Observação" }), noteInput]),
      errEl,
    ]),
    el("div", { class: "modal-actions" }, [cancelBtn, saveBtn]),
  ]);

  const backdrop = el("div", { class: "modal-backdrop" }, [card]);
  const close = () => backdrop.remove();
  backdrop.addEventListener("click", (e) => { if (e.target === backdrop) close(); });
  cancelBtn.addEventListener("click", close);

  saveBtn.addEventListener("click", async () => {
    const quantity = Number(qtyInput.value);
    if (!quantity || quantity <= 0) { errEl.textContent = "Informe uma quantidade válida."; return; }
    errEl.textContent = "";
    saveBtn.disabled = true;
    try {
      await API.moveStock({ inventoryItemId: item.id, type: currentType, quantity, note: noteInput.value.trim() });
      notify(currentType === "in" ? "Entrada registrada!" : "Saída registrada!", "success");
      close();
      onSave?.();
    } catch (err) {
      errEl.textContent = err.message || "Erro ao registrar movimentação.";
    } finally {
      saveBtn.disabled = false;
    }
  });

  document.body.appendChild(backdrop);
  renderIcons(backdrop);
  setTimeout(() => qtyInput.focus(), 80);
}

/**
 * openScanModal — atalho do leitor de código de barras. Ordem de
 * reconhecimento (ver item 6 da especificação):
 *   1. Já existe um item de estoque com esse SKU → fluxo normal de
 *      movimentação (comportamento original, inalterado).
 *   2. Não existe item, mas o código já está associado a um Pré-Produto
 *      → abre direto a entrada rápida (quantidade/validade/lote), sem
 *      pedir para escolher o produto de novo — reconhecimento automático.
 *   3. Código totalmente desconhecido → oferece associar a um
 *      Pré-Produto existente antes de prosseguir.
 */
export async function openScanModal({ code, items, depositId, onSave }) {
  const found = items.find((i) => (i.sku || "").toLowerCase() === code.toLowerCase());
  if (found) {
    openMoveModal({ item: found, onSave });
    return;
  }

  try {
    const preProduct = await API.preProductByBarcode(code);
    openEntryFromPreProductModal({ depositId, preProduct, presetBarcode: code, onSave });
    return;
  } catch {
    // Nenhum Pré-Produto associado a este código ainda — segue para associação manual.
  }

  openAssociateBarcodeModal({ code, depositId, onSave });
}

/**
 * openAssociateBarcodeModal — quando um código escaneado não corresponde
 * a nenhum item de estoque nem a nenhum Pré-Produto, permite escolher (ou
 * criar na hora, com o mesmo "+") um Pré-Produto para associar a esse
 * código. Nas próximas leituras, esse mesmo código já será reconhecido
 * automaticamente (ver openScanModal, passo 2).
 */
function openAssociateBarcodeModal({ code, depositId, onSave }) {
  const select = el("select", { class: "select" }, [el("option", { value: "", text: "Carregando…" })]);
  const addBtn = el("button", { type: "button", class: "icon-btn", title: "Novo Pré-Produto" }, [el("i", { "data-lucide": "plus" })]);
  addBtn.addEventListener("click", () => openPreProductModal({
    onSave: (created) => {
      select.appendChild(el("option", { value: created.id, text: created.name }));
      select.value = created.id;
    },
  }));

  const errEl = el("div", { class: "error-text" });
  const confirmBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "link" }), "Associar e continuar"]);
  const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });

  const card = el("div", { class: "modal" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: "Código não encontrado" })]),
    el("div", { class: "product-modal-body" }, [
      el("p", { style: "margin-bottom:14px", text: `O código "${code}" ainda não está associado a nenhum item ou Pré-Produto. Associe-o a um Pré-Produto do catálogo para continuar — nas próximas leituras ele será reconhecido automaticamente.` }),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Pré-Produto" }), el("div", { class: "field-inline-add" }, [select, addBtn])]),
      errEl,
    ]),
    el("div", { class: "modal-actions" }, [cancelBtn, confirmBtn]),
  ]);
  const backdrop = el("div", { class: "modal-backdrop" }, [card]);
  const close = () => backdrop.remove();
  backdrop.addEventListener("click", (e) => { if (e.target === backdrop) close(); });
  cancelBtn.addEventListener("click", close);

  API.preProducts().then((list) => {
    select.innerHTML = "";
    if (!list.length) {
      select.appendChild(el("option", { value: "", text: "Nenhum Pré-Produto cadastrado ainda" }));
      return;
    }
    select.appendChild(el("option", { value: "", text: "Selecione…" }));
    list.forEach((p) => select.appendChild(el("option", { value: p.id, text: p.name })));
  }).catch(() => { select.innerHTML = ""; select.appendChild(el("option", { value: "", text: "Erro ao carregar" })); });

  confirmBtn.addEventListener("click", async () => {
    if (!select.value) { errEl.textContent = "Escolha um Pré-Produto."; return; }
    errEl.textContent = "";
    confirmBtn.disabled = true;
    try {
      const associated = await API.associatePreProductBarcode(select.value, code);
      notify("Código associado ao Pré-Produto!", "success");
      close();
      openEntryFromPreProductModal({ depositId, preProduct: associated, presetBarcode: code, onSave });
    } catch (err) {
      errEl.textContent = err.message || "Erro ao associar código.";
    } finally {
      confirmBtn.disabled = false;
    }
  });

  document.body.appendChild(backdrop);
  renderIcons(backdrop);
}

/**
 * openPreProductModal — cria (ou edita) um Pré-Produto: o "molde"
 * permanente de um produto (nome, categoria, marca, unidade,
 * observações), sem nenhum dado de estoque (sem quantidade, validade,
 * lote ou código de barras — isso pertence exclusivamente ao item de
 * estoque real, criado depois a partir dele). Disponível para qualquer
 * usuário autenticado, igual à criação de itens de estoque e categorias.
 */
export async function openPreProductModal({ preProduct, onSave } = {}) {
  const isEdit = !!preProduct;

  let categories = [];
  try {
    categories = await API.categories();
  } catch {
    // Segue sem categorias pré-carregadas.
  }

  const f = {
    name: el("input", { class: "input", value: preProduct?.name || "", placeholder: "Nome *" }),
    brand: el("input", { class: "input", value: preProduct?.brand || "", placeholder: "Marca (opcional)" }),
    unit: el("input", { class: "input", value: preProduct?.unit || "", placeholder: "Unidade de medida * (ex.: kg, un, cx)" }),
    notes: el("textarea", { class: "input", rows: "3", placeholder: "Observações (opcional)", text: preProduct?.notes || "" }),
  };

  const categorySelect = el("select", { class: "select" }, [
    el("option", { value: "", text: "Nenhuma" }),
    ...categories.map((c) => el("option", { value: c.id, text: c.name, selected: preProduct?.category_id === c.id })),
  ]);
  if (preProduct?.category_id) categorySelect.value = preProduct.category_id;
  const addCategoryBtn = el("button", { type: "button", class: "icon-btn", title: "Nova categoria" }, [el("i", { "data-lucide": "plus" })]);
  addCategoryBtn.addEventListener("click", () => openCategoryModal((created) => {
    categorySelect.appendChild(el("option", { value: created.id, text: created.name }));
    categorySelect.value = created.id;
  }));
  const categoryRow = el("div", { class: "field-inline-add" }, [categorySelect, addCategoryBtn]);

  const errEl = el("div", { class: "error-text" });
  const saveBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "save" }), isEdit ? "Salvar" : "Criar"]);
  const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });

  const card = el("div", { class: "modal" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: isEdit ? "Editar Pré-Produto" : "Novo Pré-Produto" })]),
    el("div", { class: "product-modal-body" }, [
      el("p", { class: "muted", style: "font-size:0.82em;margin-bottom:14px", text: "O Pré-Produto é só um catálogo — não representa estoque. Quando o material chegar de fato, use \"Entrada com Pré-Produto\" para informar quantidade, validade e lote." }),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Nome *" }), f.name]),
      el("div", { class: "form-grid-2" }, [
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Marca" }), f.brand]),
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Unidade de medida *" }), f.unit]),
      ]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Categoria" }), categoryRow]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Observações" }), f.notes]),
      errEl,
    ]),
    el("div", { class: "modal-actions" }, [cancelBtn, saveBtn]),
  ]);

  const backdrop = el("div", { class: "modal-backdrop" }, [card]);
  const close = () => backdrop.remove();
  backdrop.addEventListener("click", (e) => { if (e.target === backdrop) close(); });
  cancelBtn.addEventListener("click", close);

  saveBtn.addEventListener("click", async () => {
    const name = f.name.value.trim();
    const unit = f.unit.value.trim();
    errEl.textContent = "";
    if (!name) { errEl.textContent = "Nome é obrigatório."; return; }
    if (!unit) { errEl.textContent = "Unidade de medida é obrigatória."; return; }

    saveBtn.disabled = true;
    try {
      const data = { name, categoryId: categorySelect.value || null, brand: f.brand.value.trim(), unit, notes: f.notes.value.trim() };
      const saved = isEdit ? await API.updatePreProduct(preProduct.id, data) : await API.createPreProduct(data);
      notify(isEdit ? "Pré-Produto atualizado!" : "Pré-Produto criado!", "success");
      close();
      onSave?.(saved);
    } catch (err) {
      errEl.textContent = err.message || "Erro ao salvar Pré-Produto.";
    } finally {
      saveBtn.disabled = false;
    }
  });

  document.body.appendChild(backdrop);
  renderIcons(backdrop);
  setTimeout(() => f.name.focus(), 80);
}

/**
 * openEntryFromPreProductModal — "Entrada utilizando Pré-Produto" (item 5
 * da especificação). Ao chegar fisicamente, o produto já cadastrado como
 * Pré-Produto é usado para criar o item de estoque real, pedindo apenas
 * os dados que só existem no momento da entrada: quantidade, validade,
 * lote e código de barras. O Pré-Produto permanece intacto no catálogo
 * para futuras entradas — nada aqui o apaga ou modifica.
 *
 * Se `preProduct` já vier definido (fluxo do scanner, código já
 * associado ou recém-associado), o seletor de produto é pulado. Se
 * `presetBarcode` vier definido, o campo de código de barras já nasce
 * preenchido com o código lido, sem precisar redigitar.
 */
export async function openEntryFromPreProductModal({ depositId, preProduct, presetBarcode, onSave }) {
  if (!depositId) { notify("Selecione um depósito antes de registrar uma entrada.", "warning", { record: false }); return; }

  let preProducts = [];
  let selectedProduct = preProduct || null;
  if (!selectedProduct) {
    try {
      preProducts = await API.preProducts();
    } catch {
      // segue com a lista vazia; o select mostrará o estado correspondente
    }
    if (!preProducts.length) {
      notify("Nenhum Pré-Produto cadastrado ainda. Cadastre um em \"Pré-Produto\" primeiro.", "warning", { record: false });
      return;
    }
  }

  const productSelect = selectedProduct ? null : el("select", { class: "select" }, [
    el("option", { value: "", text: "Selecione um Pré-Produto…" }),
    ...preProducts.map((p) => el("option", { value: p.id, text: p.brand ? `${p.name} — ${p.brand}` : p.name })),
  ]);
  const productLabel = selectedProduct ? el("p", { style: "font-weight:600", text: selectedProduct.name }) : null;

  const qtyInput = el("input", { class: "input", type: "number", min: "1", value: "1", placeholder: "Quantidade *" });
  const expiryInput = el("input", { class: "input", placeholder: "DD/MM/AAAA", inputmode: "numeric", maxlength: "10" });
  expiryInput.addEventListener("input", () => { expiryInput.value = applyDateMask(expiryInput.value); expiryErr.textContent = ""; });
  // Lote obrigatório ao entrar no estoque, inclusive vindo de Pré-Produto
  // (regra explícita da especificação — validado também no backend).
  const lotInput = el("input", { class: "input", placeholder: "Número do lote *" });
  const barcodeInput = el("input", { class: "input", value: presetBarcode || "", placeholder: "Código de barras (opcional)" });

  const expiryErr = el("div", { class: "error-text" });
  const lotErr = el("div", { class: "error-text" });
  const errEl = el("div", { class: "error-text" });
  const saveBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "check" }), "Registrar entrada"]);
  const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });

  const body = [
    el("div", { class: "field" }, [
      el("label", { class: "field-label", text: "Pré-Produto" }),
      productSelect || productLabel,
    ]),
    el("div", { class: "form-grid-2" }, [
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Quantidade *" }), qtyInput]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Data de validade *" }), expiryInput, expiryErr]),
    ]),
    el("div", { class: "form-grid-2" }, [
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Número do lote *" }), lotInput, lotErr]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Código de barras" }), barcodeInput]),
    ]),
    errEl,
  ];

  const card = el("div", { class: "modal" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: "Entrada com Pré-Produto" })]),
    el("div", { class: "product-modal-body" }, body),
    el("div", { class: "modal-actions" }, [cancelBtn, saveBtn]),
  ]);

  const backdrop = el("div", { class: "modal-backdrop" }, [card]);
  const close = () => backdrop.remove();
  backdrop.addEventListener("click", (e) => { if (e.target === backdrop) close(); });
  cancelBtn.addEventListener("click", close);

  saveBtn.addEventListener("click", async () => {
    errEl.textContent = "";
    expiryErr.textContent = "";
    lotErr.textContent = "";

    if (productSelect) {
      const chosen = preProducts.find((p) => p.id === productSelect.value);
      if (!chosen) { errEl.textContent = "Escolha um Pré-Produto."; return; }
      selectedProduct = chosen;
    }

    const quantity = Number(qtyInput.value);
    if (!quantity || quantity <= 0) { errEl.textContent = "Informe uma quantidade válida."; return; }

    const expiryValue = expiryInput.value.trim();
    if (!expiryValue || !isValidBRDate(expiryValue)) {
      expiryErr.textContent = randomFunnyDateError();
      return;
    }

    // Lote obrigatório ao entrar no estoque, mesmo vindo de um Pré-Produto
    // (o Pré-Produto em si podia não ter lote — o estoque exige).
    const lotValue = lotInput.value.trim();
    if (!lotValue) {
      lotErr.textContent = "Número do lote é obrigatório.";
      return;
    }

    saveBtn.disabled = true;
    try {
      // Marca é um campo próprio do item de estoque — nunca embutir em
      // notes (ver migração 007). Unidade continua em notes, pois
      // inventory não possui coluna de unidade própria.
      const notesParts = [];
      if (selectedProduct.unit) notesParts.push(`Unidade: ${selectedProduct.unit}`);
      if (selectedProduct.notes) notesParts.push(selectedProduct.notes);

      const createdItem = await API.createInventoryItem({
        depositId,
        name: selectedProduct.name,
        sku: barcodeInput.value.trim(),
        brand: selectedProduct.brand || "",
        minQuantity: 0,
        expiryDate: expiryValue,
        lotNumber: lotValue,
        categoryId: selectedProduct.category_id || null,
        notes: notesParts.join(" · "),
        location: {},
      });
      await API.moveStock({ inventoryItemId: createdItem.id, type: "in", quantity, note: "Entrada via Pré-Produto" });

      notify(`Entrada registrada para "${selectedProduct.name}"!`, "success");
      close();
      onSave?.();
    } catch (err) {
      errEl.textContent = err.message || "Erro ao registrar entrada.";
    } finally {
      saveBtn.disabled = false;
    }
  });

  document.body.appendChild(backdrop);
  renderIcons(backdrop);
  setTimeout(() => qtyInput.focus(), 80);
}

/**
 * openPreProductListModal — lista os Pré-Produtos cadastrados em cards,
 * com opções de Editar e Excluir (itens 4 e 5 da especificação). Reusa
 * openPreProductModal para criação/edição — não duplica o formulário.
 * A exclusão pede confirmação antes de chamar a API (evita exclusões
 * acidentais) e o backend é quem garante a integridade referencial.
 */
export async function openPreProductListModal({ onChanged } = {}) {
  const listEl = el("div", { class: "pre-product-list", style: "display:flex;flex-direction:column;gap:10px;max-height:60vh;overflow:auto" });
  const emptyEl = el("p", { class: "muted", style: "text-align:center;padding:20px 0", text: "Nenhum Pré-Produto cadastrado ainda." });
  const newBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "plus" }), "Novo Pré-Produto"]);
  const closeBtn = el("button", { class: "btn btn-ghost", text: "Fechar" });

  const card = el("div", { class: "modal modal-lg" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: "Pré-Produtos" })]),
    el("div", { class: "product-modal-body" }, [listEl]),
    el("div", { class: "modal-actions" }, [closeBtn, newBtn]),
  ]);
  const backdrop = el("div", { class: "modal-backdrop" }, [card]);
  const close = () => backdrop.remove();
  backdrop.addEventListener("click", (e) => { if (e.target === backdrop) close(); });
  closeBtn.addEventListener("click", close);

  async function reload() {
    listEl.innerHTML = "";
    let list = [];
    try {
      list = await API.preProducts();
    } catch (err) {
      listEl.appendChild(el("p", { class: "error-text", text: err.message || "Erro ao carregar Pré-Produtos." }));
      return;
    }
    if (!list.length) { listEl.appendChild(emptyEl); return; }

    list.forEach((p) => {
      const row = el("div", { class: "product-card", style: "display:flex;justify-content:space-between;align-items:center;gap:12px" }, [
        el("div", {}, [
          el("div", { class: "pc-name", text: p.name }),
          p.brand ? el("div", { class: "muted", style: "font-size:0.82em", text: `Marca: ${p.brand}` }) : el("span"),
          el("div", { class: "muted", style: "font-size:0.78em", text: [p.category?.name, p.unit].filter(Boolean).join(" · ") }),
        ]),
        el("div", { class: "pc-actions" }, [
          el("button", { class: "icon-btn", title: "Editar", onclick: guardedClick(() => openPreProductModal({
            preProduct: p,
            onSave: async () => { await reload(); onChanged?.(); },
          })) }, [el("i", { "data-lucide": "pencil" })]),
          el("button", { class: "icon-btn", title: "Excluir", onclick: guardedClick(() => confirmDeletePreProduct(p)) }, [el("i", { "data-lucide": "trash-2" })]),
        ]),
      ]);
      listEl.appendChild(row);
    });
    renderIcons(listEl);
  }

  function confirmDeletePreProduct(p) {
    // Confirmação antes de excluir, conforme item 5 da especificação —
    // evita exclusões acidentais. A checagem de integridade referencial
    // (produtos/estoque já vinculados) é responsabilidade do backend.
    openModal({
      title: "Excluir Pré-Produto",
      body: `Deseja excluir "${p.name}"${p.brand ? ` (${p.brand})` : ""}? Esta ação não pode ser desfeita.`,
      primaryLabel: "Excluir",
      danger: true,
      onConfirm: async () => {
        try {
          await API.deletePreProduct(p.id);
          notify("Pré-Produto excluído.", "warning");
          await reload();
          onChanged?.();
        } catch (err) {
          notify(err.message || "Erro ao excluir Pré-Produto.", "error");
        }
      },
    });
  }

  newBtn.addEventListener("click", () => openPreProductModal({
    onSave: async () => { await reload(); onChanged?.(); },
  }));

  document.body.appendChild(backdrop);
  await reload();
  renderIcons(backdrop);
}
