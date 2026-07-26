/**
 * Entradas e Saídas — leitor de código de barras + histórico de
 * movimentações. Toda movimentação é registrada via API.moveStock,
 * que no backend grava o StockMovement de auditoria. O histórico
 * completo nunca é apagado — o filtro de período abaixo só restringe
 * a visualização/exportação.
 */
import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { DataTable } from "../components/table.js";
import { notify } from "../components/notifications.js";
import { guardedClick } from "../utils/security.js";
import { exportExcel } from "../utils/exportExcel.js";
import { exportTxt } from "../utils/exportTxt.js";
import { BarcodeScanner } from "../components/barcodeScanner.js";
import { openMoveModal, openScanModal, openEntryFromPreProductModal } from "../components/inventoryModal.js";
import { PERIOD_PRESETS, resolvePeriod } from "../utils/period.js";

export function MovementsPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    let movementsData = [];
    let items = [];
    let deposits = [];
    let depositId = null;
    let periodPreset = "7d";
    let customFrom = "";
    let customTo = "";

    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Entradas e Saídas" }),
        el("p", { class: "muted", text: "Registre e acompanhe movimentações de estoque. Use o leitor de código de barras ou busca manual." }),
      ]),
      el("div", { class: "exports" }, [
        el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
          exportTxt(movementsData, "movimentacoes.txt"); notify("TXT exportado.", "success");
        }) }, [el("i", { "data-lucide": "file-text" }), " TXT"]),
        el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
          exportExcel(movementsData, "movimentacoes.xlsx", "Movimentações"); notify("Excel exportado.", "success");
        }) }, [el("i", { "data-lucide": "sheet" }), " Excel"]),
      ]),
    ]);

    // ── Depósito + período ───────────────────────────────────────
    const depositSelect = el("select", { class: "select", style: "max-width:260px" });
    const periodSelect = el("select", { class: "select", style: "max-width:200px" },
      PERIOD_PRESETS.map((p) => el("option", { value: p.value, text: p.label, selected: p.value === periodPreset })),
    );
    const fromInput = el("input", { type: "date", class: "input", style: "max-width:160px" });
    const toInput = el("input", { type: "date", class: "input", style: "max-width:160px" });
    const customRange = el("div", { class: "filters-row", style: "display:none;margin-top:0" }, [
      el("span", { class: "muted", text: "de" }), fromInput,
      el("span", { class: "muted", text: "até" }), toInput,
    ]);
    const filtersRow = el("div", { class: "filters-row", style: "margin-bottom:14px" }, [depositSelect, periodSelect]);

    // ── Painel do leitor de código de barras ───────────────────
    const scannerSection = el("div", { class: "card card-pad", style: "margin-bottom:18px" }, [
      el("h3", { text: "Leitor de Código de Barras", style: "margin-bottom:12px" }),
    ]);
    const scanner = BarcodeScanner({
      autoFocus: false,
      onScan: ({ code, refresh }) => handleScan(code, refresh),
    });
    scannerSection.appendChild(scanner.node);

    // ── Busca manual (sem leitor) + entrada via Pré-Produto ─────
    const manualRow = el("div", { class: "filters-row", style: "margin-bottom:18px" });
    const manualSelect = el("select", { class: "select", style: "max-width:320px" });
    manualRow.append(
      manualSelect,
      el("button", { class: "btn btn-soft", onclick: guardedClick(() => {
        const item = items.find((i) => i.id === manualSelect.value);
        if (item) openMoveModal({ item, onSave: loadMovements });
      }) }, [el("i", { "data-lucide": "arrow-left-right" }), " Movimentar item selecionado"]),
      el("button", { class: "btn btn-soft", onclick: guardedClick(() => {
        openEntryFromPreProductModal({ depositId, onSave: loadMovements });
      }) }, [el("i", { "data-lucide": "package-plus" }), " Entrada com Pré-Produto"]),
    );

    const tableContainer = el("div", {}, [
      el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Carregando movimentações…"]),
    ]);

    content.append(head, filtersRow, customRange, scannerSection, manualRow, tableContainer);
    renderIcons(content);

    function handleScan(code, refreshHistory) {
      refreshHistory?.();
      openScanModal({ code, items, depositId, onSave: loadMovements });
    }

    function itemName(id) {
      return items.find((i) => i.id === id)?.name || id.slice(0, 8) + "…";
    }

    function currentRange() {
      if (periodPreset === "custom") {
        return { from: customFrom || undefined, to: customTo || undefined };
      }
      return resolvePeriod(periodPreset) || {};
    }

    function renderTable(rows) {
      tableContainer.innerHTML = "";
      if (!rows.length) {
        tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhuma movimentação encontrada para o período selecionado."]));
        return;
      }
      const table = DataTable({
        rows, pageSize: 10,
        columns: [
          { key: "tipo", label: "Tipo", render: (r) =>
            el("span", { class: `chip ${r.tipo === "entrada" ? "chip-success" : "chip-warning"}`, text: r.tipo })
          },
          { key: "item", label: "Item" },
          { key: "quantidade", label: "Qtd" },
          { key: "data", label: "Data" },
          { key: "observacao", label: "Observação" },
        ],
      });
      tableContainer.appendChild(table.node);
    }

    function renderDepositOptions() {
      depositSelect.innerHTML = "";
      deposits.forEach((d) => depositSelect.appendChild(el("option", { value: d.id, text: d.name })));
      depositSelect.value = depositId;
      depositSelect.style.display = deposits.length > 1 ? "" : "none";
    }

    function renderManualOptions() {
      manualSelect.innerHTML = "";
      manualSelect.appendChild(el("option", { value: "", text: "Selecione um item…" }));
      items.forEach((i) => manualSelect.appendChild(el("option", { value: i.id, text: `${i.name} (saldo: ${i.quantity})` })));
    }

    async function loadMovements() {
      try {
        deposits = await API.deposits({ scope: "stock", classId: session.classId });
        if (!deposits.length) {
          tableContainer.innerHTML = "";
          tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhum depósito de estoque disponível."]));
          return;
        }
        depositId = session.depositId && deposits.some((d) => d.id === session.depositId)
          ? session.depositId
          : deposits[0].id;
        session.setDepositId(depositId);
        renderDepositOptions();

        const { from, to } = currentRange();
        const [movements, inventoryItems] = await Promise.all([
          API.movements({ depositId, classId: session.classId, from, to }),
          API.inventory(depositId, session.classId),
        ]);
        items = inventoryItems;
        renderManualOptions();

        movementsData = movements.map((m) => ({
          id: m.id,
          tipo: m.type === "in" ? "entrada" : "saida",
          item: itemName(m.inventory_item_id),
          quantidade: m.quantity ?? "—",
          data: m.created_at ? m.created_at.slice(0, 10) : "—",
          observacao: m.note || "—",
        }));

        renderTable(movementsData);
      } catch (err) {
        tableContainer.innerHTML = "";
        tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Erro ao carregar movimentações."]));
        notify(err.message || "Erro.", "error");
      }
    }

    depositSelect.addEventListener("change", (e) => {
      depositId = e.target.value;
      session.setDepositId(depositId);
      loadMovements();
    });

    periodSelect.addEventListener("change", (e) => {
      periodPreset = e.target.value;
      customRange.style.display = periodPreset === "custom" ? "flex" : "none";
      if (periodPreset !== "custom") loadMovements();
    });
    fromInput.addEventListener("change", (e) => { customFrom = e.target.value; if (periodPreset === "custom") loadMovements(); });
    toInput.addEventListener("change", (e) => { customTo = e.target.value; if (periodPreset === "custom") loadMovements(); });

    loadMovements();
  });
}
