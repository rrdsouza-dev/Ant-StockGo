import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { DataTable } from "../components/table.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { exportExcel } from "../utils/exportExcel.js";
import { exportTxt } from "../utils/exportTxt.js";
import { notify } from "../components/notifications.js";
import { guardedClick } from "../utils/security.js";
import { PERIOD_PRESETS, resolvePeriod } from "../utils/period.js";

export function ReportsPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    let movementsData = [];
    let deposits = [];
    let items = [];
    let depositId = "";
    let periodPreset = "30d";
    let customFrom = "";
    let customTo = "";
    let selectedRows = [];
    let table = null;

    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Relatórios" }),
        el("p", { class: "muted", text: "Histórico de movimentações de estoque registradas no sistema. O histórico completo nunca é apagado — os filtros só restringem a visualização." }),
      ]),
    ]);

    // ── Exportação: "todos os listados" (sempre disponível) e
    // "selecionados" (só aparece quando há alguma linha marcada) ──────
    const exportAllBtns = el("div", { class: "exports" }, [
      el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
        exportTxt(movementsData, "movimentacoes.txt");
        notify("TXT exportado (todos os listados).", "success");
      }) }, [el("i", { "data-lucide": "file-text" }), "TXT"]),
      el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
        exportExcel(movementsData, "movimentacoes.xlsx", "Movimentações");
        notify("Excel exportado (todos os listados).", "success");
      }) }, [el("i", { "data-lucide": "sheet" }), "Excel"]),
    ]);
    const selectedCount = el("span", { class: "muted", style: "font-size:0.85em" });
    const exportSelectedBtns = el("div", { class: "exports", style: "display:none" }, [
      selectedCount,
      el("button", { class: "btn btn-soft", onclick: guardedClick(() => {
        exportTxt(selectedRows, "movimentacoes-selecionadas.txt");
        notify("TXT exportado (selecionados).", "success");
      }) }, [el("i", { "data-lucide": "file-text" }), "TXT selecionados"]),
      el("button", { class: "btn btn-soft", onclick: guardedClick(() => {
        exportExcel(selectedRows, "movimentacoes-selecionadas.xlsx", "Movimentações");
        notify("Excel exportado (selecionados).", "success");
      }) }, [el("i", { "data-lucide": "sheet" }), "Excel selecionados"]),
    ]);
    head.appendChild(el("div", {}, [exportAllBtns, exportSelectedBtns]));

    // ── Filtros: depósito + período ────────────────────────────────
    const depositSelect = el("select", { class: "select", style: "max-width:280px" }, [
      el("option", { value: "", text: "Todos os depósitos acessíveis" }),
    ]);
    const periodSelect = el("select", { class: "select", style: "max-width:220px" },
      PERIOD_PRESETS.map((p) => el("option", { value: p.value, text: p.label, selected: p.value === periodPreset })),
    );
    const fromInput = el("input", { type: "date", class: "input", style: "max-width:170px" });
    const toInput = el("input", { type: "date", class: "input", style: "max-width:170px" });
    const customRange = el("div", { class: "filters-row", style: "display:none;margin-top:0" }, [
      el("span", { class: "muted", text: "de" }), fromInput,
      el("span", { class: "muted", text: "até" }), toInput,
    ]);
    const filtersRow = el("div", { class: "filters-row", style: "margin-bottom:14px" }, [depositSelect, periodSelect]);

    const tableContainer = el("div", {}, [
      el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Carregando movimentações..."]),
    ]);

    content.append(head, filtersRow, customRange, tableContainer);
    renderIcons(content);

    function itemName(id) {
      return items.find((i) => i.id === id)?.name || (id || "—").slice(0, 8) + "…";
    }

    function currentRange() {
      if (periodPreset === "custom") {
        return { from: customFrom || undefined, to: customTo || undefined };
      }
      return resolvePeriod(periodPreset) || {};
    }

    async function loadReports() {
      try {
        deposits = await API.deposits({ scope: "stock", classId: session.classId });
        depositSelect.innerHTML = "";
        depositSelect.appendChild(el("option", { value: "", text: "Todos os depósitos acessíveis" }));
        deposits.forEach((d) => depositSelect.appendChild(el("option", { value: d.id, text: d.name })));
        depositSelect.value = depositId;

        const { from, to } = currentRange();

        const [movements, allItems] = await Promise.all([
          API.movements({ depositId: depositId || undefined, classId: session.classId, from, to }),
          API.inventory(depositId || undefined, session.classId),
        ]);
        items = allItems;
        movementsData = movements.map((m) => normalizeMovement(m));
        selectedRows = [];
        updateSelectedUI();

        tableContainer.innerHTML = "";
        if (movementsData.length === 0) {
          table = null;
          tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhuma movimentação encontrada para o período selecionado."]));
          return;
        }

        table = DataTable({
          rows: movementsData,
          pageSize: 8,
          selectable: true,
          onSelectionChange: (rows) => { selectedRows = rows; updateSelectedUI(); },
          columns: [
            { key: "id", label: "ID", render: (r) => el("span", { class: "muted", style: "font-size:0.78em", text: r.id ? r.id.slice(0, 8) + "…" : "—" }) },
            { key: "tipo", label: "Tipo", render: (r) => {
              const cls = r.tipo === "entrada" ? "chip-success" : "chip-warning";
              return el("span", { class: `chip ${cls}`, text: r.tipo });
            }},
            { key: "item", label: "Item" },
            { key: "quantidade", label: "Quantidade" },
            { key: "data", label: "Data" },
            { key: "observacao", label: "Observação" },
          ],
        });
        tableContainer.appendChild(table.node);
      } catch (err) {
        tableContainer.innerHTML = "";
        tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Erro ao carregar movimentações."]));
        notify(err.message || "Erro ao carregar relatórios.", "error");
      }
    }

    function updateSelectedUI() {
      const n = selectedRows.length;
      exportAllBtns.style.display = n > 0 ? "none" : "flex";
      exportSelectedBtns.style.display = n > 0 ? "flex" : "none";
      selectedCount.textContent = `${n} selecionado(s)`;
    }

    function normalizeMovement(m) {
      return {
        id: m.id,
        tipo: m.type === "in" ? "entrada" : "saida",
        item: itemName(m.inventory_item_id),
        quantidade: m.quantity ?? "—",
        data: m.created_at ? m.created_at.slice(0, 10) : "—",
        observacao: m.note || "—",
      };
    }

    depositSelect.addEventListener("change", (e) => {
      depositId = e.target.value;
      loadReports();
    });

    periodSelect.addEventListener("change", (e) => {
      periodPreset = e.target.value;
      customRange.style.display = periodPreset === "custom" ? "flex" : "none";
      if (periodPreset !== "custom") loadReports();
    });
    fromInput.addEventListener("change", (e) => { customFrom = e.target.value; if (periodPreset === "custom") loadReports(); });
    toInput.addEventListener("change", (e) => { customTo = e.target.value; if (periodPreset === "custom") loadReports(); });

    loadReports();
  });
}
