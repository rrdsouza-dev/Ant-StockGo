import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { exportExcel } from "../utils/exportExcel.js";
import { exportTxt } from "../utils/exportTxt.js";
import { notify } from "../components/notifications.js";
import { DataTable } from "../components/table.js";
import { guardedClick } from "../utils/security.js";

export function DashboardPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    let movementsData = [];

    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Visão Geral" }),
        el("p", { class: "muted", text: "Panorama do estoque e das movimentações recentes." }),
      ]),
    ]);

    const exportRow = el("div", { class: "filters-row" }, [
      el("div", { class: "exports" }, [
        el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
          exportTxt(movementsData, "movimentacoes.txt");
          notify("Arquivo TXT exportado.", "success");
        }) }, [el("i", { "data-lucide": "file-text" }), "Exportar TXT"]),
        el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
          exportExcel(movementsData, "movimentacoes.xlsx", "Movimentações");
          notify("Planilha Excel exportada.", "success");
        }) }, [el("i", { "data-lucide": "sheet" }), "Exportar Excel"]),
      ]),
    ]);

    const statTotal = el("div", { class: "value", text: "—" });
    const statEntradas = el("div", { class: "value", text: "—" });
    const statBaixo = el("div", { class: "value", text: "—" });
    const statDepositos = el("div", { class: "value", text: "—" });

    const stats = el("div", { class: "stat-grid" }, [
      statCardNode("Itens em Estoque", statTotal, "package"),
      statCardNode("Entradas nos Últimos 7 Dias", statEntradas, "check-circle-2"),
      statCardNode("Itens Abaixo do Mínimo", statBaixo, "alert-triangle"),
      statCardNode("Depósitos Acessíveis", statDepositos, "warehouse"),
    ]);

    const pieCard = chartCard("Distribuição de Estoque por Depósito",
      el("div", { class: "chart-box pie" }, [el("canvas", { id: "chart-pie" })]),
      el("div", { class: "legend", id: "pie-legend" }, []),
    );
    const barCard = chartCard("Movimentações Recentes (Entradas vs Saídas)",
      el("div", { class: "chart-box" }, [el("canvas", { id: "chart-bar" })]),
    );
    const dash = el("div", { class: "dash-grid" }, [pieCard, barCard]);

    const tableWrap = el("div", { style: "margin-top:18px" }, [
      el("div", { style: "display:flex;align-items:center;justify-content:space-between;margin-bottom:10px" }, [
        el("h2", { text: "Movimentações Recentes" }),
      ]),
      el("div", { id: "movements-table-container" }, [
        el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Carregando dados..."]),
      ]),
    ]);

    content.append(head, exportRow, stats, dash, tableWrap);
    renderIcons(content);

    async function loadDashboard() {
      try {
        const deposits = await API.deposits({ scope: "stock", classId: session.classId });
        if (!deposits.length) {
          notify("Nenhum depósito de estoque disponível para este usuário.", "warning", { record: false });
          updateStats([], [], deposits);
          renderMovementsTable([]);
          return;
        }
        statDepositos.textContent = deposits.length.toLocaleString("pt-BR");

        const [items, movements] = await Promise.all([
          API.inventory(undefined, session.classId).catch(() => []),
          API.movements({ classId: session.classId }).catch(() => []),
        ]);

        movementsData = movements.map((m) => normalizeMovement(m, items));

        updateStats(items, movements, deposits);
        renderMovementsTable(movementsData);

        setTimeout(() => drawCharts(items, movements, deposits), 0);
      } catch (err) {
        notify(err.message || "Erro ao carregar dashboard.", "error");
        renderMovementsTable([]);
      }
    }

    function updateStats(items, movements, deposits) {
      const totalItems = items.reduce((s, i) => s + (i.quantity || 0), 0);
      statTotal.textContent = totalItems.toLocaleString("pt-BR");

      const sevenDaysAgo = Date.now() - 7 * 24 * 60 * 60 * 1000;
      const entradas = (movements || []).filter((m) => m.type === "in" && new Date(m.created_at).getTime() >= sevenDaysAgo).length;
      statEntradas.textContent = entradas.toLocaleString("pt-BR");

      const baixo = items.filter((i) => i.quantity <= i.min_quantity).length;
      statBaixo.textContent = baixo.toLocaleString("pt-BR");

      statDepositos.textContent = deposits.length.toLocaleString("pt-BR");
    }

    function renderMovementsTable(rows) {
      const container = document.getElementById("movements-table-container");
      if (!container) return;
      container.innerHTML = "";
      if (rows.length === 0) {
        container.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhuma movimentação encontrada."]));
        return;
      }
      const table = DataTable({
        rows,
        pageSize: 6,
        columns: [
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
      container.appendChild(table.node);
    }

    loadDashboard();
  });
}

function normalizeMovement(m, items) {
  const item = items.find((i) => i.id === m.inventory_item_id);
  return {
    tipo: m.type === "in" ? "entrada" : "saida",
    item: item ? item.name : (m.inventory_item_id || "—").slice(0, 8) + "…",
    quantidade: m.quantity ?? "—",
    data: m.created_at ? m.created_at.slice(0, 10) : "—",
    observacao: m.note || "—",
  };
}

function statCardNode(label, valueEl, icon) {
  return el("div", { class: "stat-card" }, [
    el("div", { class: "icon-pill" }, [el("i", { "data-lucide": icon })]),
    el("div", { class: "label", text: label }),
    valueEl,
  ]);
}

function chartCard(title, ...children) {
  return el("div", { class: "card chart-card" }, [
    el("div", { class: "chart-header" }, [
      el("h3", { text: title }),
      el("button", { class: "icon-btn", title: "Mais" }, [el("i", { "data-lucide": "more-horizontal" })]),
    ]),
    ...children,
  ]);
}

function drawCharts(items, movements, deposits) {
  if (!window.Chart) return;

  const depositMap = {};
  for (const d of deposits) depositMap[d.id] = d.name;

  const stockByDeposit = {};
  for (const i of items) {
    const name = depositMap[i.deposit_id] || "Depósito";
    stockByDeposit[name] = (stockByDeposit[name] || 0) + (i.quantity || 0);
  }
  const depLabels = Object.keys(stockByDeposit);
  const depValues = depLabels.map((k) => stockByDeposit[k]);
  const palette = ["#10b981", "#7ed29c", "#bde9cb", "#6366f1", "#f59e0b", "#ef4444", "#3b82f6"];

  const pie = document.getElementById("chart-pie");
  if (pie) {
    new Chart(pie, {
      type: "doughnut",
      data: {
        labels: depLabels.length ? depLabels : ["Sem dados"],
        datasets: [{ data: depValues.length ? depValues : [1], backgroundColor: palette.slice(0, depLabels.length || 1), borderWidth: 0 }],
      },
      options: { plugins: { legend: { display: false } }, cutout: "55%", responsive: true, maintainAspectRatio: false },
    });

    const legend = document.getElementById("pie-legend");
    if (legend) {
      legend.innerHTML = "";
      depLabels.forEach((l, i) => {
        const span = el("span", {}, []);
        const dot = el("i", { style: `background:${palette[i % palette.length]};width:10px;height:10px;border-radius:50%;display:inline-block;margin-right:6px` });
        span.appendChild(dot);
        span.appendChild(document.createTextNode(l));
        legend.appendChild(span);
      });
    }
  }

  const today = new Date();
  const dayLabels = [];
  const entryData = [];
  const exitData = [];
  for (let i = 6; i >= 0; i--) {
    const d = new Date(today);
    d.setDate(d.getDate() - i);
    const key = d.toISOString().slice(0, 10);
    dayLabels.push(key.slice(5));
    const dayMov = movements.filter((m) => (m.created_at || "").slice(0, 10) === key);
    entryData.push(dayMov.filter((m) => m.type === "in").reduce((s, m) => s + m.quantity, 0));
    exitData.push(dayMov.filter((m) => m.type === "out").reduce((s, m) => s + m.quantity, 0));
  }

  const bar = document.getElementById("chart-bar");
  if (bar) {
    new Chart(bar, {
      type: "bar",
      data: {
        labels: dayLabels,
        datasets: [
          { label: "Entradas", data: entryData, backgroundColor: "#10b981", borderRadius: 8, barThickness: 22 },
          { label: "Saídas", data: exitData, backgroundColor: "#bde9cb", borderRadius: 8, barThickness: 22 },
        ],
      },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: { legend: { display: true, position: "top" } },
        scales: {
          x: { grid: { display: false } },
          y: { ticks: { callback: (v) => (v >= 1000 ? v / 1000 + "K" : v) }, grid: { color: "#eef3f0" } },
        },
      },
    });
  }
}
