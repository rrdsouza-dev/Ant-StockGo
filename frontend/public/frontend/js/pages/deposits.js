/**
 * Depósitos — CRUD de estoques físicos/lógicos da escola.
 * Entidade central do sistema: todo item de inventário pertence a um
 * depósito, e todo acesso de professor é resolvido através das turmas
 * vinculadas a estes depósitos.
 */
import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { DataTable } from "../components/table.js";
import { notify } from "../components/notifications.js";
import { openModal } from "../components/modal.js";
import { guardedClick } from "../utils/security.js";

export function DepositsPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    const isGestao = session.user?.role === "gestao";

    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Depósitos" }),
        el("p", { class: "muted", text: "Estoques da escola. Cada item de inventário pertence a um destes depósitos." }),
      ]),
      isGestao
        ? el("button", { class: "btn btn-soft", onclick: guardedClick(() => openDepositModal(null)) }, [
            el("i", { "data-lucide": "plus" }), "Novo depósito",
          ])
        : el("span"),
    ]);

    const tableContainer = el("div", {}, [
      el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Carregando depósitos…"]),
    ]);
    content.append(head, tableContainer);
    renderIcons(content);

    async function load() {
      try {
        const deposits = await API.deposits();
        tableContainer.innerHTML = "";
        if (!deposits.length) {
          tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhum depósito cadastrado."]));
          return;
        }
        const rows = deposits.map((d) => ({ ...d, data_fmt: d.created_at?.slice(0, 10) || "—" }));
        const table = DataTable({
          rows, pageSize: 8,
          columns: [
            { key: "name", label: "Nome" },
            { key: "description", label: "Descrição", render: (r) => el("span", { text: r.description || "—" }) },
            { key: "is_administrative", label: "Uso", render: (r) =>
              r.is_administrative
                ? el("span", { class: "chip chip-warning", text: "Administrativo" })
                : el("span", { class: "muted", text: "Turma" })
            },
            { key: "data_fmt", label: "Criado em" },
            ...(isGestao ? [{
              key: "acoes", label: "Ações", render: (r) =>
                el("div", { class: "pc-actions" }, [
                  el("button", { class: "icon-btn", title: "Editar", onclick: guardedClick(() => openDepositModal(r)) }, [el("i", { "data-lucide": "pencil" })]),
                  el("button", { class: "icon-btn", title: "Desativar", onclick: guardedClick(() => deleteDeposit(r)) }, [el("i", { "data-lucide": "trash-2" })]),
                ])
            }] : []),
          ],
        });
        tableContainer.appendChild(table.node);
        renderIcons(tableContainer);
      } catch (err) {
        tableContainer.innerHTML = "";
        tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Erro ao carregar depósitos."]));
        notify(err.message || "Erro.", "error");
      }
    }

    function openDepositModal(deposit) {
      const isEdit = !!deposit;
      const f = {
        name: el("input", { class: "input", value: deposit?.name || "", placeholder: "Nome do depósito *" }),
        desc: el("input", { class: "input", value: deposit?.description || "", placeholder: "Descrição (opcional)" }),
        isAdmin: el("input", { type: "checkbox", checked: !!deposit?.is_administrative }),
      };
      const errEl = el("div", { class: "error-text" });
      const saveBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "save" }), isEdit ? "Salvar" : "Criar"]);
      const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });

      const card = el("div", { class: "modal" }, [
        el("div", { class: "modal-header" }, [el("h3", { text: isEdit ? "Editar depósito" : "Novo depósito" })]),
        el("div", { class: "product-modal-body" }, [
          el("div", { class: "field" }, [el("label", { class: "field-label", text: "Nome *" }), f.name]),
          el("div", { class: "field" }, [el("label", { class: "field-label", text: "Descrição" }), f.desc]),
          el("div", { class: "field" }, [
            el("label", { style: "display:flex;align-items:center;gap:8px;cursor:pointer" }, [
              f.isAdmin,
              el("span", { text: "Depósito administrativo (uso exclusivo da gestão)" }),
            ]),
            el("p", { class: "muted", style: "font-size:0.78em;margin-top:4px", text: "Só pode haver um depósito administrativo por vez — marcar este desmarca automaticamente o anterior. É o único depósito cujo estoque a gestão acessa; os demais são acessados pelas turmas vinculadas." }),
          ]),
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
          const data = { name, description: f.desc.value.trim(), isAdministrative: f.isAdmin.checked };
          if (isEdit) await API.updateDeposit(deposit.id, data);
          else await API.createDeposit(data);
          notify(isEdit ? "Depósito atualizado!" : "Depósito criado!", "success");
          close(); load();
        } catch (err) {
          errEl.textContent = err.message || "Erro ao salvar depósito.";
        } finally {
          saveBtn.disabled = false;
        }
      });

      document.body.appendChild(backdrop);
      renderIcons(backdrop);
      setTimeout(() => f.name.focus(), 80);
    }

    function deleteDeposit(deposit) {
      openModal({
        title: "Desativar depósito",
        body: `Desativar "${deposit.name}"? Os itens de estoque e o histórico continuam preservados.`,
        primaryLabel: "Desativar", danger: true,
        onConfirm: async () => {
          await API.deleteDeposit(deposit.id);
          notify("Depósito desativado.", "warning");
          load();
        },
      });
    }

    load();
  });
}
