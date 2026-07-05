/**
 * Usuários — painel da gestão com duas abas:
 *  - "Pendentes": contas aguardando aprovação (fluxo PENDENTE -> ativo).
 *  - "Ativos": contas já aprovadas no sistema.
 * Novas contas nascem sempre como PENDENTE em /auth/register; só viram
 * usuários reais quando um gestor aprova aqui.
 */
import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { API, ROLE_LABEL } from "../services/api.js";
import { DataTable } from "../components/table.js";
import { notify } from "../components/notifications.js";
import { openModal } from "../components/modal.js";
import { guardedClick } from "../utils/security.js";

export function UsersPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    let active = "pending";

    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Usuários" }),
        el("p", { class: "muted", text: "Aprove novas contas e acompanhe quem já tem acesso ao sistema." }),
      ]),
    ]);

    const tabs = [
      { key: "pending", label: "Pendentes" },
      { key: "active", label: "Ativos" },
    ];
    const tabsRow = el("div", { class: "tabs" });
    const body = el("div");

    function renderTabs() {
      tabsRow.innerHTML = "";
      tabs.forEach((t) => {
        const b = el("button", { class: "tab" + (active === t.key ? " active" : ""), text: t.label });
        b.addEventListener("click", () => { active = t.key; renderTabs(); loadBody(); });
        tabsRow.appendChild(b);
      });
    }

    async function loadBody() {
      body.innerHTML = "";
      body.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Carregando…"]));
      try {
        if (active === "pending") await renderPending();
        else await renderActive();
      } catch (err) {
        body.innerHTML = "";
        body.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Erro ao carregar."]));
        notify(err.message || "Erro ao carregar usuários.", "error");
      }
    }

    async function renderPending() {
      const pending = await API.pendingUsers();
      body.innerHTML = "";
      if (!pending.length) {
        body.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhuma conta aguardando aprovação."]));
        return;
      }
      const table = DataTable({
        rows: pending, pageSize: 10, search: false,
        columns: [
          { key: "name", label: "Nome" },
          { key: "email", label: "Email" },
          { key: "role", label: "Perfil", render: (r) =>
            el("span", { class: `chip ${r.role === "gestao" ? "chip-warning" : "chip-success"}`, text: ROLE_LABEL[r.role] || r.role })
          },
          { key: "requested_at", label: "Solicitado em", render: (r) => el("span", { text: r.requested_at ? r.requested_at.slice(0, 10) : "—" }) },
          { key: "acoes", label: "Ações", render: (r) =>
            el("div", { class: "pc-actions" }, [
              el("button", { class: "btn btn-soft", onclick: guardedClick(() => decide(r, "approve")) }, [el("i", { "data-lucide": "check" }), " Aprovar"]),
              el("button", { class: "btn btn-ghost", onclick: guardedClick(() => decide(r, "reject")) }, [el("i", { "data-lucide": "x" }), " Rejeitar"]),
            ])
          },
        ],
      });
      body.appendChild(table.node);
      renderIcons(body);
    }

    function decide(pendingUser, action) {
      const isApprove = action === "approve";
      openModal({
        title: isApprove ? "Aprovar conta" : "Rejeitar conta",
        body: isApprove
          ? `Aprovar "${pendingUser.name}"? A conta será ativada e poderá entrar no sistema imediatamente.`
          : `Rejeitar "${pendingUser.name}"? O cadastro será descartado.`,
        primaryLabel: isApprove ? "Aprovar" : "Rejeitar",
        danger: !isApprove,
        onConfirm: async () => {
          await API.approveUser(pendingUser.id, action);
          notify(isApprove ? "Conta aprovada!" : "Conta rejeitada.", isApprove ? "success" : "warning");
          loadBody();
        },
      });
    }

    async function renderActive() {
      const users = await API.users();
      body.innerHTML = "";
      if (!users.length) {
        body.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhum usuário ativo."]));
        return;
      }
      const table = DataTable({
        rows: users, pageSize: 10,
        columns: [
          { key: "name", label: "Nome" },
          { key: "email", label: "Email" },
          { key: "role", label: "Perfil", render: (r) =>
            el("span", { class: `chip ${r.role === "gestao" ? "chip-warning" : "chip-success"}`, text: ROLE_LABEL[r.role] || r.role })
          },
          { key: "classes", label: "Turmas", render: (r) =>
            r.classes?.length
              ? el("div", { style: "display:flex;flex-wrap:wrap;gap:4px" }, r.classes.map((c) => el("span", { class: "perm-badge", text: c.name })))
              : el("span", { class: "muted", text: "—" })
          },
          { key: "active", label: "Status", render: (r) =>
            el("span", { class: `chip ${r.active ? "chip-success" : "chip-danger"}`, text: r.active ? "Ativo" : "Inativo" })
          },
        ],
      });
      body.appendChild(table.node);
      renderIcons(body);
    }

    content.append(head, tabsRow, body);
    renderIcons(content);
    renderTabs();
    loadBody();
  });
}
