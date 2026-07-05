/**
 * inventoryModal.js — Modais de item de estoque e de movimentação
 * (entrada/saída). Substitui o antigo productModal.js: não existe mais
 * "produto isolado" — todo modal aqui opera sobre um item de inventário
 * pertencente a um depósito.
 */
import { el, renderIcons } from "../utils/helpers.js";
import { API } from "../services/api.js";
import { notify } from "./notifications.js";

/**
 * openInventoryItemModal — cria ou edita um item de estoque.
 * Fluxo: chama API.createInventoryItem / API.updateInventoryItem.
 * Restrito à gestão na UI (o backend também recusaria para professor).
 */
export function openInventoryItemModal({ depositId, item, onSave }) {
  const isEdit = !!item;
  const f = {
    name: el("input", { class: "input", value: item?.name || "", placeholder: "Nome do item *" }),
    sku: el("input", { class: "input", value: item?.sku || "", placeholder: "Código / SKU (opcional)" }),
    min: el("input", { class: "input", type: "number", min: "0", value: item?.min_quantity ?? 0, placeholder: "Quantidade mínima" }),
  };
  const errEl = el("div", { class: "error-text" });
  const saveBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "save" }), isEdit ? "Salvar" : "Criar"]);
  const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });

  const card = el("div", { class: "modal" }, [
    el("div", { class: "modal-header" }, [el("h3", { text: isEdit ? "Editar item de estoque" : "Novo item de estoque" })]),
    el("div", { class: "product-modal-body" }, [
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Nome *" }), f.name]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Código / SKU" }), f.sku]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Quantidade mínima" }), f.min]),
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
    if (!name) { errEl.textContent = "Nome é obrigatório."; return; }
    errEl.textContent = "";
    saveBtn.disabled = true;
    try {
      const data = { name, sku: f.sku.value.trim(), minQuantity: Number(f.min.value) || 0 };
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
