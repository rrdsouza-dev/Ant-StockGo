/**
 * Entradas e Saídas — leitor de código de barras + histórico de
 * movimentações. Toda movimentação é registrada via API.moveStock,
 * que no backend grava o StockMovement de auditoria.
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
import { openMoveModal, openScanModal } from "../components/inventoryModal.js";

export function MovementsPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    let movementsData = [];
    let items = [];
    let deposits = [];
    let depositId = null;

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

    const depositSelect = el("select", { class: "select", style: "max-width:260px;margin-bottom:14px" });

    // ── Painel do leitor de código de barras ───────────────────
    const scannerSection = el("div", { class: "card card-pad", style: "margin-bottom:18px" }, [
      el("h3", { text: "Leitor de Código de Barras", style: "margin-bottom:12px" }),
    ]);
    const scanner = BarcodeScanner({
      autoFocus: false,
      onScan: ({ code, refresh }) => handleScan(code, refresh),
    });
    scannerSection.appendChild(scanner.node);

    // ── Busca manual (sem leitor) ───────────────────────────────
    const manualRow = el("div", { class: "filters-row", style: "margin-bottom:18px" });
    const manualSelect = el("select", { class: "select", style: "max-width:320px" });
    manualRow.append(
      manualSelect,
      el("button", { class: "btn btn-soft", onclick: guardedClick(() => {
        const item = items.find((i) => i.id === manualSelect.value);
        if (item) openMoveModal({ item, onSave: loadMovements });
      }) }, [el("i", { "data-lucide": "arrow-left-right" }), " Movimentar item selecionado"]),
    );

    const tableContainer = el("div", {}, [
      el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Carregando movimentações…"]),
    ]);

    content.append(head, depositSelect, scannerSection, manualRow, tableContainer);
    renderIcons(content);

    function handleScan(code, refreshHistory) {
      refreshHistory?.();
      openScanModal({ code, items, onSave: loadMovements });
    }

    function itemName(id) {
      return items.find((i) => i.id === id)?.name || id.slice(0, 8) + "…";
    }

    function renderTable(rows) {
      tableContainer.innerHTML = "";
      if (!rows.length) {
        tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhuma movimentação registrada."]));
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

        const [movements, inventoryItems] = await Promise.all([
          API.movements({ depositId, classId: session.classId }),
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

    loadMovements();
  });
}
