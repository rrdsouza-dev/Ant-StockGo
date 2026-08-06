/**
 * Estoque — itens de inventário de um depósito, com saldo e quantidade
 * mínima. Não existe "produto isolado": todo item pertence sempre a um
 * depósito e sua quantidade só muda através de movimentações auditadas.
 */
import { el, renderIcons, debounce } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { notify } from "../components/notifications.js";
import { guardedClick, sanitize } from "../utils/security.js";
import { exportExcel } from "../utils/exportExcel.js";
import { exportTxt } from "../utils/exportTxt.js";
import { openInventoryItemModal, openMoveModal, openPreProductListModal } from "../components/inventoryModal.js";
import { openModal } from "../components/modal.js";

/** Converte "2026-12-31" (ou ISO completo) para "31/12/2026" para exibição. */
function isoToBRDate(iso) {
  if (!iso) return "";
  const datePart = String(iso).slice(0, 10);
  const [y, m, d] = datePart.split("-");
  if (!y || !m || !d) return "";
  return `${d}/${m}/${y}`;
}

/** Monta um texto curto "Corredor 3 · Torre 2 · Prateleira 5 · A7" a partir do objeto location. */
function formatLocation(loc) {
  if (!loc) return "";
  const parts = [];
  if (loc.aisle) parts.push(`Corredor ${loc.aisle}`);
  if (loc.tower) parts.push(`Torre ${loc.tower}`);
  if (loc.shelf) parts.push(`Prateleira ${loc.shelf}`);
  if (loc.position) parts.push(loc.position);
  return parts.join(" · ");
}

/** Marca como "vencendo" itens com validade nos próximos 30 dias ou já vencidos. */
function isExpiringSoon(iso) {
  if (!iso) return false;
  const expiry = new Date(String(iso).slice(0, 10));
  const in30Days = new Date();
  in30Days.setDate(in30Days.getDate() + 30);
  return expiry.getTime() <= in30Days.getTime();
}

function truncate(text, max) {
  if (!text || text.length <= max) return text || "";
  return text.slice(0, max - 1) + "…";
}

export function InventoryPage(root, ctx) {
  let query = "";
  let items = [];
  let deposits = [];
  let depositId = null;

  AppShell(root, ctx.path, (content) => {
    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Estoque" }),
        el("p", { class: "muted", text: "Itens de inventário do depósito selecionado e seus saldos atuais." }),
      ]),
      el("div", { class: "exports" }, [
        el("button", { class: "btn btn-soft", onclick: guardedClick(() => {
          if (!depositId) { notify("Selecione um depósito.", "warning", { record: false }); return; }
          openInventoryItemModal({ depositId, onSave: load });
        }) }, [el("i", { "data-lucide": "plus" }), "Adicionar item"]),
        el("button", { class: "btn btn-soft", onclick: guardedClick(() => openPreProductListModal({ onChanged: load })) }, [el("i", { "data-lucide": "package-plus" }), "Pré-Produtos"]),
        el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
          exportTxt(exportRows(), "estoque.txt");
          notify("Exportado TXT.", "success");
        }) }, [el("i", { "data-lucide": "file-text" }), "TXT"]),
        el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
          exportExcel(exportRows(), "estoque.xlsx", "Estoque");
          notify("Exportado Excel.", "success");
        }) }, [el("i", { "data-lucide": "sheet" }), "Excel"]),
      ]),
    ]);

    const search = el("input", { class: "input", placeholder: "Buscar por nome ou código…", style: "max-width:340px;position:relative;top:-6px" });
    const depositSelect = el("select", { class: "select", style: "max-width:260px" });
    const filters = el("div", { class: "filters-row" }, [search, depositSelect]);
    const grid = el("div", { class: "product-grid stagger" });

    content.append(head, filters, grid);
    renderIcons(content);

    function exportRows() {
      return visibleItems().map((i) => ({
        nome: i.name,
        marca: i.brand || "",
        sku: i.sku || "",
        quantidade: i.quantity,
        quantidade_minima: i.min_quantity,
        validade: isoToBRDate(i.expiry_date),
        lote: i.lot_number || "",
        categoria: i.category?.name || "",
        localizacao: formatLocation(i.location),
        observacoes: i.notes || "",
      }));
    }

    function visibleItems() {
      const q = sanitize(query).toLowerCase();
      if (!q) return items;
      return items.filter((i) => [i.name, i.sku, i.brand].some((v) => (v || "").toLowerCase().includes(q)));
    }

    function statusFor(item) {
      if (item.quantity === 0) return { label: "Esgotado", cls: "chip-danger" };
      if (item.quantity <= item.min_quantity) return { label: "Estoque baixo", cls: "chip-warning" };
      return { label: "Em estoque", cls: "chip-success" };
    }

    // Conta quantos itens de estoque compartilham o mesmo código de
    // barras (SKU). Quando > 1, é o mesmo produto (ou produtos
    // diferentes que por acaso têm o mesmo código) guardado em lotes
    // separados — usado para mostrar o indicador verde no card.
    function countBySku() {
      const counts = {};
      items.forEach((i) => {
        const sku = (i.sku || "").trim().toLowerCase();
        if (!sku) return;
        counts[sku] = (counts[sku] || 0) + 1;
      });
      return counts;
    }

    function rerender() {
      grid.innerHTML = "";
      const rows = visibleItems();
      if (!rows.length) {
        grid.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center;grid-column:1/-1" }, ["Nenhum item de estoque encontrado."]));
        return;
      }
      const skuCounts = countBySku();
      rows.forEach((item) => grid.appendChild(buildCard(item, skuCounts)));
      renderIcons(grid);
    }

    function buildCard(item, skuCounts) {
      const status = statusFor(item);
      const locationText = formatLocation(item.location);
      const expiryText = isoToBRDate(item.expiry_date);
      const expiryWarn = isExpiringSoon(item.expiry_date);
      const sku = (item.sku || "").trim().toLowerCase();
      const hasMultipleLots = sku && (skuCounts[sku] || 0) > 1;

      const metaLine = el("div", { class: "pc-meta" });
      if (item.category?.name) metaLine.appendChild(el("span", { class: "chip chip-info", text: item.category.name }));
      if (expiryText) metaLine.appendChild(el("span", { class: `chip ${expiryWarn ? "chip-danger" : "chip-success"}`, text: `Validade: ${expiryText}` }));
      // Lote em destaque (badge), já que produtos iguais podem ter vários
      // lotes simultâneos com estoque/validade próprios (ver item 16/17
      // da especificação). Cor sinaliza se está perto de vencer.
      if (item.lot_number) {
        metaLine.appendChild(el("span", {
          class: `chip ${expiryWarn ? "chip-danger" : "chip-info"}`,
          style: "font-weight:600",
          text: `Lote ${item.lot_number}`,
        }));
      }
      // Quantidade mínima em destaque no card — já existia como dado
      // (usado no cálculo de "Estoque baixo"), agora também fica visível.
      metaLine.appendChild(el("span", { class: "muted", style: "font-size:0.78em", text: `Qtd. mínima: ${item.min_quantity}` }));

      const card = el("article", { class: "product-card", style: "position:relative" }, [
        // Indicador de "separado por lote": aparece quando este código de
        // barras tem mais de um item de estoque cadastrado (mesmo produto
        // em lotes diferentes, ou produtos/marcas diferentes que
        // compartilham o código). Só um sinal visual rápido — o card
        // continua mostrando os dados do lote específico normalmente.
        hasMultipleLots ? el("span", {
          title: `Este código tem ${skuCounts[sku]} lotes cadastrados`,
          style: "position:absolute;top:-4px;right:-4px;width:12px;height:12px;border-radius:50%;background:#22c55e;border:2px solid #fff;box-shadow:0 0 0 1px rgba(34,197,94,0.4)",
        }) : el("span"),
        el("div", { class: "pc-head" }, [
          el("div", {}, [
            el("div", { class: "pc-name", text: item.name }),
            // Marca logo abaixo do nome, bem clara — permite diferenciar
            // produtos com nome/código parecidos mas marcas diferentes,
            // sem depender de observações (ver itens 2/3/7 da especificação
            // original e o complemento de marca no card).
            item.brand ? el("div", { class: "muted", style: "font-size:0.85em;font-weight:600", text: `Marca: ${item.brand}` }) : el("span"),
            el("div", { class: "pc-code", text: `Código: ${item.sku || item.id.slice(0, 8)}` }),
          ]),
          el("span", { class: `chip ${status.cls}`, text: status.label }),
        ]),
        metaLine,
        locationText ? el("div", { class: "location-badge" }, [el("i", { "data-lucide": "map-pin" }), locationText]) : el("span"),
        item.notes ? el("p", { class: "muted", style: "font-size:0.8em;margin-top:6px", title: item.notes, text: truncate(item.notes, 80) }) : el("span"),
        el("div", { class: "pc-row" }, [
          el("div", { class: "pc-qty", html: `${item.quantity}<span>un</span>` }),
          el("div", { class: "pc-actions" }, [
            el("button", { class: "icon-btn", title: "Movimentar", onclick: guardedClick(() => openMoveModal({ item, onSave: load })) }, [el("i", { "data-lucide": "arrow-left-right" })]),
            el("button", { class: "icon-btn", title: "Editar", onclick: guardedClick(() => openInventoryItemModal({ depositId, item, onSave: load })) }, [el("i", { "data-lucide": "pencil" })]),
            el("button", { class: "icon-btn", title: "Desativar", onclick: guardedClick(() => confirmDelete(item)) }, [el("i", { "data-lucide": "trash-2" })]),
          ]),
        ]),
      ]);
      renderIcons(card);
      return card;
    }

    function confirmDelete(item) {
      openModal({
        title: "Desativar item",
        body: `Deseja desativar "${item.name}"? O histórico de movimentações é preservado.`,
        primaryLabel: "Desativar",
        danger: true,
        onConfirm: async () => {
          await API.deleteInventoryItem(item.id);
          notify("Item desativado.", "warning");
          load();
        },
      });
    }

    function renderDepositOptions() {
      depositSelect.innerHTML = "";
      deposits.forEach((d) => depositSelect.appendChild(el("option", { value: d.id, text: d.name })));
      depositSelect.value = depositId;
      depositSelect.style.display = deposits.length > 1 ? "" : "none";
    }

    async function load() {
      grid.innerHTML = "";
      grid.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center;grid-column:1/-1" }, ["Carregando estoque…"]));
      try {
        deposits = await API.deposits({ scope: "stock", classId: session.classId });
        if (!deposits.length) {
          grid.innerHTML = "";
          grid.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center;grid-column:1/-1" }, ["Nenhum depósito de estoque disponível para este usuário."]));
          return;
        }
        depositId = session.depositId && deposits.some((d) => d.id === session.depositId)
          ? session.depositId
          : deposits[0].id;
        session.setDepositId(depositId);
        renderDepositOptions();

        items = await API.inventory(depositId, session.classId);
        rerender();
      } catch (err) {
        grid.innerHTML = "";
        grid.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center;grid-column:1/-1" }, ["Erro ao carregar o estoque."]));
        notify(err.message || "Erro ao carregar estoque.", "error");
      }
    }

    search.addEventListener("input", debounce((e) => { query = e.target.value; rerender(); }, 200));
    depositSelect.addEventListener("change", (e) => {
      depositId = e.target.value;
      session.setDepositId(depositId);
      load();
    });

    load();
  });
}
