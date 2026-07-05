import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { DataTable } from "../components/table.js";
import { API } from "../services/api.js";
import { exportExcel } from "../utils/exportExcel.js";
import { exportTxt } from "../utils/exportTxt.js";
import { notify } from "../components/notifications.js";
import { guardedClick } from "../utils/security.js";

export function ReportsPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    let movementsData = [];
    let deposits = [];
    let items = [];
    let depositId = "";

    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Relatórios" }),
        el("p", { class: "muted", text: "Histórico de movimentações de estoque registradas no sistema." }),
      ]),
      el("div", { class: "exports" }, [
        el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
          exportTxt(movementsData, "movimentacoes.txt");
          notify("TXT exportado.", "success");
        }) }, [el("i", { "data-lucide": "file-text" }), "TXT"]),
        el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
          exportExcel(movementsData, "movimentacoes.xlsx", "Movimentações");
          notify("Excel exportado.", "success");
        }) }, [el("i", { "data-lucide": "sheet" }), "Excel"]),
      ]),
    ]);

    const depositSelect = el("select", { class: "select", style: "max-width:280px;margin-bottom:14px" }, [
      el("option", { value: "", text: "Todos os depósitos acessíveis" }),
    ]);

    const tableContainer = el("div", {}, [
      el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Carregando movimentações..."]),
    ]);

    content.append(head, depositSelect, tableContainer);
    renderIcons(content);

    function itemName(id) {
      return items.find((i) => i.id === id)?.name || (id || "—").slice(0, 8) + "…";
    }

    async function loadReports() {
      try {
        deposits = await API.deposits();
        depositSelect.innerHTML = "";
        depositSelect.appendChild(el("option", { value: "", text: "Todos os depósitos acessíveis" }));
        deposits.forEach((d) => depositSelect.appendChild(el("option", { value: d.id, text: d.name })));
        depositSelect.value = depositId;

        const [movements, allItems] = await Promise.all([
          API.movements({ depositId: depositId || undefined }),
          API.inventory(depositId || undefined),
        ]);
        items = allItems;
        movementsData = movements.map((m) => normalizeMovement(m));

        tableContainer.innerHTML = "";
        if (movementsData.length === 0) {
          tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhuma movimentação registrada."]));
          return;
        }

        const table = DataTable({
          rows: movementsData,
          pageSize: 8,
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

    loadReports();
  });
}
