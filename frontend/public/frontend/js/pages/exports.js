import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { exportExcel } from "../utils/exportExcel.js";
import { exportTxt } from "../utils/exportTxt.js";
import { notify } from "../components/notifications.js";
import { guardedClick } from "../utils/security.js";

export function ExportsPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    const isGestao = session.user?.role === "gestao";

    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Exportações" }),
        el("p", { class: "muted", text: "Exporte os dados reais do sistema para uso externo." }),
      ]),
    ]);

    const grid = el("div", { class: "product-grid" });
    const statusMsg = el("div", { class: "muted", style: "padding:20px;text-align:center" }, ["Carregando dados..."]);
    grid.appendChild(statusMsg);

    content.append(head, grid);
    renderIcons(content);

    async function loadExports() {
      try {
        const deposits = await API.deposits({ scope: "stock", classId: session.classId });
        if (!deposits.length) {
          grid.innerHTML = "";
          grid.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhum depósito de estoque disponível para este usuário."]));
          return;
        }

        const [items, movements, classes, allDeposits] = await Promise.all([
          API.inventory(undefined, session.classId).catch(() => []),
          API.movements({ classId: session.classId }).catch(() => []),
          API.classes().catch(() => []),
          isGestao ? API.deposits().catch(() => []) : Promise.resolve([]),
        ]);

        const depositMap = {};
        for (const d of deposits) depositMap[d.id] = d.name;
        const itemMap = {};
        for (const i of items) itemMap[i.id] = i.name;

        const dataSets = [
          {
            key: "depositos",
            label: "Depósitos",
            rows: allDeposits.map((d) => ({
              id: d.id,
              nome: d.name,
              descricao: d.description || "",
              administrativo: d.is_administrative ? "sim" : "não",
              criado_em: d.created_at ? d.created_at.slice(0, 10) : "",
            })),
          },
          {
            key: "estoque",
            label: "Estoque (itens de inventário)",
            rows: items.map((i) => ({
              id: i.id,
              nome: i.name,
              sku: i.sku || "",
              deposito: depositMap[i.deposit_id] || "",
              quantidade: i.quantity,
              quantidade_minima: i.min_quantity,
              validade: i.expiry_date ? i.expiry_date.slice(0, 10).split("-").reverse().join("/") : "",
              lote: i.lot_number || "",
              categoria: i.category?.name || "",
              localizacao: [
                i.location?.aisle ? `Corredor ${i.location.aisle}` : "",
                i.location?.tower ? `Torre ${i.location.tower}` : "",
                i.location?.shelf ? `Prateleira ${i.location.shelf}` : "",
                i.location?.position || "",
              ].filter(Boolean).join(" · "),
              observacoes: i.notes || "",
              atualizado_em: i.updated_at ? i.updated_at.slice(0, 10) : "",
            })),
          },
          {
            key: "movimentacoes",
            label: "Movimentações de estoque",
            rows: movements.map((m) => ({
              id: m.id,
              tipo: m.type === "in" ? "entrada" : "saida",
              item: itemMap[m.inventory_item_id] || m.inventory_item_id,
              deposito: depositMap[m.deposit_id] || "",
              quantidade: m.quantity,
              observacao: m.note || "",
              data: m.created_at ? m.created_at.slice(0, 10) : "",
            })),
          },
          {
            key: "turmas",
            label: "Turmas",
            rows: classes.map((c) => ({
              id: c.id,
              nome: c.name,
              descricao: c.description || "",
              depositos: (c.deposits || []).map((d) => d.name).join(", "),
              professores_vinculados: c.teacher_ids?.length || 0,
            })),
          },
        ];

        grid.innerHTML = "";
        for (const ds of dataSets) {
          if (!isGestao && (ds.key === "depositos" || ds.key === "turmas")) continue;
          grid.appendChild(el("div", { class: "card card-pad" }, [
            el("h3", { text: ds.label }),
            el("p", { class: "muted", style: "margin:6px 0 16px", text: `${ds.rows.length} registro(s) disponíveis para exportação.` }),
            el("div", { style: "display:flex;gap:10px;flex-wrap:wrap" }, [
              el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
                exportTxt(ds.rows, `${ds.key}.txt`);
                notify("TXT exportado.", "success");
              }) }, [el("i", { "data-lucide": "file-text" }), "TXT"]),
              el("button", { class: "btn btn-primary", onclick: guardedClick(() => {
                exportExcel(ds.rows, `${ds.key}.xlsx`, ds.label);
                notify("Excel exportado.", "success");
              }) }, [el("i", { "data-lucide": "sheet" }), "Excel"]),
            ]),
          ]));
        }
        renderIcons(grid);
      } catch (err) {
        grid.innerHTML = "";
        grid.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Erro ao carregar dados para exportação."]));
        notify(err.message || "Erro ao carregar exportações.", "error");
      }
    }

    loadExports();
  });
}
