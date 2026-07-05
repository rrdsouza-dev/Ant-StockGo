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
import { openInventoryItemModal, openMoveModal } from "../components/inventoryModal.js";
import { openModal } from "../components/modal.js";

export function InventoryPage(root, ctx) {
  let query = "";
  let items = [];
  let deposits = [];
  let depositId = null;
  const isGestao = session.user?.role === "gestao";

  AppShell(root, ctx.path, (content) => {
    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Estoque" }),
        el("p", { class: "muted", text: "Itens de inventário do depósito selecionado e seus saldos atuais." }),
      ]),
      el("div", { class: "exports" }, [
        isGestao
          ? el("button", { class: "btn btn-soft", onclick: guardedClick(() => {
              if (!depositId) { notify("Selecione um depósito.", "warning"); return; }
              openInventoryItemModal({ depositId, onSave: load });
            }) }, [el("i", { "data-lucide": "plus" }), "Adicionar item"])
          : el("span"),
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

    const search = el("input", { class: "input", placeholder: "Buscar por nome ou código…", style: "max-width:340px" });
    const depositSelect = el("select", { class: "select", style: "max-width:260px" });
    const filters = el("div", { class: "filters-row" }, [search, depositSelect]);
    const grid = el("div", { class: "product-grid stagger" });

    content.append(head, filters, grid);
    renderIcons(content);

    function exportRows() {
      return visibleItems().map((i) => ({
        nome: i.name, sku: i.sku || "", quantidade: i.quantity, quantidade_minima: i.min_quantity,
      }));
    }

    function visibleItems() {
      const q = sanitize(query).toLowerCase();
      if (!q) return items;
      return items.filter((i) => [i.name, i.sku].some((v) => (v || "").toLowerCase().includes(q)));
    }

    function statusFor(item) {
      if (item.quantity === 0) return { label: "Esgotado", cls: "chip-danger" };
      if (item.quantity <= item.min_quantity) return { label: "Estoque baixo", cls: "chip-warning" };
      return { label: "Em estoque", cls: "chip-success" };
    }

    function rerender() {
      grid.innerHTML = "";
      const rows = visibleItems();
      if (!rows.length) {
        grid.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center;grid-column:1/-1" }, ["Nenhum item de estoque encontrado."]));
        return;
      }
      rows.forEach((item) => grid.appendChild(buildCard(item)));
      renderIcons(grid);
    }

    function buildCard(item) {
      const status = statusFor(item);
      const card = el("article", { class: "product-card" }, [
        el("div", { class: "pc-head" }, [
          el("div", {}, [
            el("div", { class: "pc-code", text: item.sku || item.id.slice(0, 8) }),
            el("div", { class: "pc-name", text: item.name }),
          ]),
          el("span", { class: `chip ${status.cls}`, text: status.label }),
        ]),
        el("div", { class: "pc-row" }, [
          el("div", { class: "pc-qty", html: `${item.quantity}<span>un</span>` }),
          el("div", { class: "pc-actions" }, [
            el("button", { class: "icon-btn", title: "Movimentar", onclick: guardedClick(() => openMoveModal({ item, onSave: load })) }, [el("i", { "data-lucide": "arrow-left-right" })]),
            ...(isGestao ? [
              el("button", { class: "icon-btn", title: "Editar", onclick: guardedClick(() => openInventoryItemModal({ depositId, item, onSave: load })) }, [el("i", { "data-lucide": "pencil" })]),
              el("button", { class: "icon-btn", title: "Desativar", onclick: guardedClick(() => confirmDelete(item)) }, [el("i", { "data-lucide": "trash-2" })]),
            ] : []),
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
        deposits = await API.deposits();
        if (!deposits.length) {
          grid.innerHTML = "";
          grid.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center;grid-column:1/-1" }, ["Nenhum depósito vinculado ao usuário."]));
          return;
        }
        depositId = session.depositId && deposits.some((d) => d.id === session.depositId)
          ? session.depositId
          : deposits[0].id;
        session.setDepositId(depositId);
        renderDepositOptions();

        items = await API.inventory(depositId);
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
