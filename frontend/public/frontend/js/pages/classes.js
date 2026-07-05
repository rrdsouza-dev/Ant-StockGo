/**
 * Turmas — criadas pela gestão, vinculam professores a depósitos.
 * Um professor só acessa/movimenta os depósitos das turmas às quais
 * está vinculado; esta tela é onde esse vínculo é definido.
 */
import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { DataTable } from "../components/table.js";
import { notify } from "../components/notifications.js";
import { openModal } from "../components/modal.js";
import { guardedClick } from "../utils/security.js";

export function ClassesPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    const isGestao = session.user?.role === "gestao";
    let teachers = [];
    let deposits = [];

    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Turmas" }),
        el("p", { class: "muted", text: isGestao
          ? "Crie turmas e vincule professores aos depósitos que podem acessar."
          : "Turmas às quais você está vinculado e os depósitos que elas liberam." }),
      ]),
      isGestao
        ? el("button", { class: "btn btn-soft", onclick: guardedClick(() => openClassModal(null)) }, [el("i", { "data-lucide": "plus" }), "Nova turma"])
        : el("span"),
    ]);

    const tableContainer = el("div", {}, [
      el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Carregando turmas…"]),
    ]);
    content.append(head, tableContainer);
    renderIcons(content);

    async function load() {
      try {
        const [classes, users, deps] = await Promise.all([
          API.classes(),
          isGestao ? API.users() : Promise.resolve([]),
          isGestao ? API.deposits() : Promise.resolve([]),
        ]);
        teachers = users.filter((u) => u.role === "professor");
        deposits = deps;

        tableContainer.innerHTML = "";
        if (!classes.length) {
          tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhuma turma cadastrada."]));
          return;
        }

        const table = DataTable({
          rows: classes, pageSize: 8,
          columns: [
            { key: "name", label: "Turma" },
            { key: "description", label: "Descrição", render: (r) => el("span", { text: r.description || "—" }) },
            { key: "deposits", label: "Depósitos vinculados", render: (r) =>
              r.deposits?.length
                ? el("div", { style: "display:flex;flex-wrap:wrap;gap:4px" }, r.deposits.map((d) => el("span", { class: "perm-badge", text: d.name })))
                : el("span", { class: "muted", text: "—" })
            },
            { key: "teacher_ids", label: "Professores", render: (r) => el("span", { class: "muted", text: `${r.teacher_ids?.length || 0} vinculado(s)` }) },
            ...(isGestao ? [{
              key: "acoes", label: "Ações", render: (r) =>
                el("div", { class: "pc-actions" }, [
                  el("button", { class: "icon-btn", title: "Editar", onclick: guardedClick(() => openClassModal(r)) }, [el("i", { "data-lucide": "pencil" })]),
                  el("button", { class: "icon-btn", title: "Remover", onclick: guardedClick(() => deleteClass(r)) }, [el("i", { "data-lucide": "trash-2" })]),
                ]),
            }] : []),
          ],
        });
        tableContainer.appendChild(table.node);
        renderIcons(tableContainer);
      } catch (err) {
        tableContainer.innerHTML = "";
        tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Erro ao carregar turmas."]));
        notify(err.message || "Erro.", "error");
      }
    }

    function multiSelect(options, selectedIds, labelKey = "name") {
      const wrap = el("div", { class: "perfil-selector", style: "flex-wrap:wrap" });
      const state = new Set(selectedIds);
      options.forEach((opt) => {
        const btn = el("button", {
          type: "button",
          class: "btn-perfil" + (state.has(opt.id) ? " active" : ""),
          text: opt[labelKey],
        });
        btn.addEventListener("click", () => {
          if (state.has(opt.id)) state.delete(opt.id); else state.add(opt.id);
          btn.classList.toggle("active");
        });
        wrap.appendChild(btn);
      });
      return { node: wrap, get: () => Array.from(state) };
    }

    function openClassModal(cls) {
      const isEdit = !!cls;
      const nameInput = el("input", { class: "input", value: cls?.name || "", placeholder: "Nome da turma *" });
      const descInput = el("input", { class: "input", value: cls?.description || "", placeholder: "Descrição (opcional)" });

      const teacherPicker = teachers.length
        ? multiSelect(teachers, cls?.teacher_ids || [])
        : { node: el("p", { class: "muted", text: "Nenhum professor ativo cadastrado ainda." }), get: () => [] };
      const depositPicker = deposits.length
        ? multiSelect(deposits, (cls?.deposits || []).map((d) => d.id))
        : { node: el("p", { class: "muted", text: "Nenhum depósito cadastrado ainda." }), get: () => [] };

      const errEl = el("div", { class: "error-text" });
      const saveBtn = el("button", { class: "btn btn-primary" }, [el("i", { "data-lucide": "save" }), isEdit ? "Salvar" : "Criar"]);
      const cancelBtn = el("button", { class: "btn btn-ghost", text: "Cancelar" });

      const card = el("div", { class: "modal" }, [
        el("div", { class: "modal-header" }, [el("h3", { text: isEdit ? "Editar turma" : "Nova turma" })]),
        el("div", { class: "product-modal-body" }, [
          el("div", { class: "field" }, [el("label", { class: "field-label", text: "Nome *" }), nameInput]),
          el("div", { class: "field" }, [el("label", { class: "field-label", text: "Descrição" }), descInput]),
          el("div", { class: "field" }, [el("label", { class: "field-label", text: "Professores vinculados" }), teacherPicker.node]),
          el("div", { class: "field" }, [el("label", { class: "field-label", text: "Depósitos que a turma libera" }), depositPicker.node]),
          errEl,
        ]),
        el("div", { class: "modal-actions" }, [cancelBtn, saveBtn]),
      ]);

      const backdrop = el("div", { class: "modal-backdrop" }, [card]);
      const close = () => backdrop.remove();
      backdrop.addEventListener("click", (e) => { if (e.target === backdrop) close(); });
      cancelBtn.addEventListener("click", close);

      saveBtn.addEventListener("click", async () => {
        const name = nameInput.value.trim();
        if (!name) { errEl.textContent = "Nome é obrigatório."; return; }
        errEl.textContent = "";
        saveBtn.disabled = true;
        try {
          const data = {
            name,
            description: descInput.value.trim(),
            teacherIds: teacherPicker.get(),
            depositIds: depositPicker.get(),
          };
          if (isEdit) await API.updateClass(cls.id, data);
          else await API.createClass(data);
          notify(isEdit ? "Turma atualizada!" : "Turma criada!", "success");
          close(); load();
        } catch (err) {
          errEl.textContent = err.message || "Erro ao salvar turma.";
        } finally {
          saveBtn.disabled = false;
        }
      });

      document.body.appendChild(backdrop);
      renderIcons(backdrop);
      setTimeout(() => nameInput.focus(), 80);
    }

    function deleteClass(cls) {
      openModal({
        title: "Remover turma",
        body: `Remover a turma "${cls.name}"? Os professores perderão o acesso aos depósitos vinculados apenas por ela.`,
        primaryLabel: "Remover", danger: true,
        onConfirm: async () => {
          await API.deleteClass(cls.id);
          notify("Turma removida.", "warning");
          load();
        },
      });
    }

    load();
  });
}
