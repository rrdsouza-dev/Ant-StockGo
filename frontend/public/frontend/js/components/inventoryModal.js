/**
 * inventoryModal.js — Modais de item de estoque e de movimentação
 * (entrada/saída). Substitui o antigo productModal.js: não existe mais
 * "produto isolado" — todo modal aqui opera sobre um item de inventário
 * pertencente a um depósito.
 */
import { el, renderIcons } from "../utils/helpers.js";
import { API } from "../services/api.js";
import { notify } from "./notifications.js";
import { applyDateMask, isValidBRDate, randomFunnyDateError } from "../utils/validators.js";

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
    min: el("input", { class: "input", type: "number", min: "0", value: item?.min_quantity ?? 0, placeholder: "Quantidade mínima" }),
    expiry: el("input", {
      class: "input", value: isoToBRDate(item?.expiry_date), placeholder: "DD/MM/AAAA", inputmode: "numeric", maxlength: "10",
    }),
    lot: el("input", { class: "input", value: item?.lot_number || "", placeholder: "Número do lote (opcional)" }),
    notes: el("textarea", { class: "input", rows: "3", placeholder: "Observações (opcional)", text: item?.notes || "" }),
  };
  f.expiry.addEventListener("input", () => { f.expiry.value = applyDateMask(f.expiry.value); expiryErr.textContent = ""; });

  // ── Categoria + botão "+" ──────────────────────────────────
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
  const errEl = el("div", { class: "error-text" });
  const saveBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "save" }), isEdit ? "Salvar" : "Criar"]);
  const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });

  const card = el("div", { class: "modal modal-lg" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: isEdit ? "Editar item de estoque" : "Novo item de estoque" })]),
    el("div", { class: "product-modal-body" }, [
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Nome *" }), f.name]),
      el("div", { class: "form-grid-2" }, [
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Código / SKU" }), f.sku]),
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Quantidade mínima" }), f.min]),
      ]),
      el("div", { class: "form-grid-2" }, [
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Data de validade *" }), f.expiry, expiryErr]),
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Número do lote" }), f.lot]),
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
    if (!name) { errEl.textContent = "Nome é obrigatório."; return; }

    const expiryValue = f.expiry.value.trim();
    if (!expiryValue || !isValidBRDate(expiryValue)) {
      expiryErr.textContent = randomFunnyDateError();
      return;
    }

    saveBtn.disabled = true;
    try {
      const data = {
        name,
        sku: f.sku.value.trim(),
        minQuantity: Number(f.min.value) || 0,
        expiryDate: expiryValue,
        lotNumber: f.lot.value.trim(),
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
function openCategoryModal(onCreated) {
  const nameInput = el("input", { class: "input", placeholder: "Nome da categoria *" });
  const errEl = el("div", { class: "error-text" });
  const saveBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "save" }), "Criar"]);
  const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });

  const card = el("div", { class: "modal modal-sm" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: "Nova categoria" })]),
    el("div", { class: "product-modal-body" }, [
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Nome *" }), nameInput]),
      errEl,
    ]),
    el("div", { class: "modal-actions" }, [cancelBtn, saveBtn]),
  ]);
  const backdrop = el("div", { class: "modal-backdrop" }, [card]);
  const close = () => backdrop.remove();
  backdrop.addEventListener("click", (e) => { if (e.target === backdrop) close(); });
  cancelBtn.addEventListener("click", close);

  saveBtn.addEventListener("click", async () => {
    const name = nameInput.value.trim();
    if (!name) { errEl.textContent = "Nome é obrigatório."; return; }
    saveBtn.disabled = true;
    try {
      const created = await API.createCategory(name);
      notify("Categoria criada!", "success");
      close();
      onCreated?.(created);
    } catch (err) {
      errEl.textContent = err.message || "Erro ao criar categoria.";
    } finally {
      saveBtn.disabled = false;
    }
  });

  document.body.appendChild(backdrop);
  renderIcons(backdrop);
  setTimeout(() => nameInput.focus(), 80);
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
 * openScanModal — atalho do leitor de código de barras: procura o item
 * pelo SKU lido e, se encontrado, abre direto o fluxo de movimentação.
 * Se não encontrado, apenas informa — não cria nada automaticamente.
 */
export function openScanModal({ code, items, onSave }) {
  const found = items.find((i) => (i.sku || "").toLowerCase() === code.toLowerCase());
  if (found) {
    openMoveModal({ item: found, onSave });
    return;
  }

  const card = el("div", { class: "modal" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: "Código não encontrado" })]),
    el("div", { class: "product-modal-body" }, [
      el("p", { text: `Nenhum item de estoque está cadastrado com o código "${code}".` }),
    ]),
    el("div", { class: "modal-actions" }, [
      el("button", { class: "btn btn-primary", text: "Fechar", onclick: () => backdrop.remove() }),
    ]),
  ]);
  const backdrop = el("div", { class: "modal-backdrop" }, [card]);
  backdrop.addEventListener("click", (e) => { if (e.target === backdrop) backdrop.remove(); });
  document.body.appendChild(backdrop);
  renderIcons(backdrop);
}
