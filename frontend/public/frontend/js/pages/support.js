/**
 * support.js — Suporte.
 *
 * Professor: abre chamados (categoria, tipo, código pessoal, descrição).
 * Nome e e-mail nunca são digitados — vêm sempre da sessão autenticada.
 * O código é validado exclusivamente no backend (ver SupportService);
 * aqui só exibimos a mensagem que o servidor devolve.
 *
 * Gestão: visualiza todos os chamados, exporta cada um individualmente
 * para TXT, e pode limpar todo o histórico mediante código administrativo
 * (também validado só no backend).
 */
import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { notify } from "../components/notifications.js";
import { openModal } from "../components/modal.js";
import { DataTable } from "../components/table.js";
import { guardedClick } from "../utils/security.js";
import { exportTxt } from "../utils/exportTxt.js";

const CATEGORIAS = [
  { value: "estoque", label: "Estoque" },
  { value: "sistema", label: "Sistema" },
];
const TIPOS = [
  { value: "erro_operacional", label: "Erro Operacional" },
  { value: "erro_sistematico", label: "Erro Sistemático" },
];

export function SupportPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    const isGestao = session.user?.role === "gestao";
    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Suporte" }),
        el("p", { class: "muted", text: isGestao
          ? "Chamados abertos pelos professores."
          : "Abra um chamado para a gestão. Nome e e-mail são preenchidos automaticamente." }),
      ]),
    ]);
    content.append(head);
    renderIcons(content);

    if (isGestao) renderGestaoView(content);
    else renderProfessorForm(content);
  });
}

function renderProfessorForm(content) {
  const user = session.user || {};

  const nameField = el("input", { class: "input", value: user.name || "", disabled: true });
  const emailField = el("input", { class: "input", value: user.email || "", disabled: true });
  const categoriaSelect = el("select", { class: "select" }, CATEGORIAS.map((c) => el("option", { value: c.value, text: c.label })));
  const tipoSelect = el("select", { class: "select" }, TIPOS.map((t) => el("option", { value: t.value, text: t.label })));
  const codigoInput = el("input", { class: "input", placeholder: "XXX-XXX-XXX" });
  const descricaoInput = el("textarea", { class: "input", rows: "4", placeholder: "Fala paê" });

  const errEl = el("div", { class: "error-text" });
  const submitBtn = el("button", { class: "btn btn-primary", type: "submit" }, [el("i", { "data-lucide": "send" }), "Enviar chamado"]);

  const form = el("form", { class: "card card-pad" }, [
    el("div", { class: "form-grid-2" }, [
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Nome" }), nameField]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Email" }), emailField]),
    ]),
    el("div", { class: "form-grid-2" }, [
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Categoria" }), categoriaSelect]),
      el("div", { class: "field" }, [el("label", { class: "field-label", text: "Tipo" }), tipoSelect]),
    ]),
    el("div", { class: "field" }, [
      el("label", { class: "field-label", text: "Código do professor *" }),
      codigoInput,
      el("p", { class: "muted", style: "font-size:0.78em;margin-top:4px", text: "Encontre seu código na tela de Perfil." }),
    ]),
    el("div", { class: "field" }, [el("label", { class: "field-label", text: "Descrição" }), descricaoInput]),
    errEl,
    el("div", { style: "display:flex;justify-content:flex-end;margin-top:10px" }, [submitBtn]),
  ]);

  form.addEventListener("submit", guardedClick(async (e) => {
    e.preventDefault();
    errEl.textContent = "";
    submitBtn.disabled = true;
    try {
      await API.createSupportTicket({
        categoria: categoriaSelect.value,
        tipo: tipoSelect.value,
        codigo: codigoInput.value.trim(),
        descricao: descricaoInput.value.trim(),
      });
      notify("Chamado enviado com sucesso!", "success");
      codigoInput.value = "";
      descricaoInput.value = "";
    } catch (err) {
      // O backend devolve aqui as mensagens exigidas ("Tá errado mano.",
      // "JÁ FALEI QUE TA ERRADO MALUCO KKKK." ou o aviso de bloqueio
      // temporário) — nunca validamos o código no frontend.
      errEl.textContent = err.message || "Erro ao enviar chamado.";
    } finally {
      submitBtn.disabled = false;
    }
  }));

  content.appendChild(form);
  renderIcons(content);
}

function renderGestaoView(content) {
  const tableContainer = el("div", { style: "margin-top:6px" }, [
    el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Carregando chamados…"]),
  ]);
  const clearBtn = el("button", { class: "btn btn-ghost", onclick: guardedClick(() => promptClearHistory()) }, [
    el("i", { "data-lucide": "trash-2" }), "Limpar histórico",
  ]);
  content.appendChild(el("div", { style: "display:flex;justify-content:flex-end;margin-bottom:10px" }, [clearBtn]));
  content.appendChild(tableContainer);
  renderIcons(content);

  const categoriaLabel = (v) => CATEGORIAS.find((c) => c.value === v)?.label || v;
  const tipoLabel = (v) => TIPOS.find((t) => t.value === v)?.label || v;

  async function load() {
    try {
      const tickets = await API.supportTickets();
      tableContainer.innerHTML = "";
      if (!tickets.length) {
        tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Nenhum chamado registrado."]));
        return;
      }
      const table = DataTable({
        rows: tickets, pageSize: 10,
        columns: [
          { key: "nome", label: "Professor" },
          { key: "email", label: "Email" },
          { key: "categoria", label: "Categoria", render: (r) => el("span", { class: "chip chip-info", text: categoriaLabel(r.categoria) }) },
          { key: "tipo", label: "Tipo", render: (r) => el("span", { class: "chip chip-warning", text: tipoLabel(r.tipo) }) },
          { key: "descricao", label: "Descrição", render: (r) => el("span", { title: r.descricao, text: r.descricao ? (r.descricao.length > 60 ? r.descricao.slice(0, 59) + "…" : r.descricao) : "—" }) },
          { key: "created_at", label: "Data", render: (r) => el("span", { text: r.created_at ? r.created_at.slice(0, 10) : "—" }) },
          { key: "acoes", label: "Ações", render: (r) =>
            el("button", { class: "icon-btn", title: "Exportar TXT", onclick: guardedClick(() => exportTicket(r)) }, [el("i", { "data-lucide": "file-text" })])
          },
        ],
      });
      tableContainer.appendChild(table.node);
      renderIcons(tableContainer);
    } catch (err) {
      tableContainer.innerHTML = "";
      tableContainer.appendChild(el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Erro ao carregar chamados."]));
      notify(err.message || "Erro.", "error");
    }
  }

  function exportTicket(ticket) {
    exportTxt([{
      professor: ticket.nome,
      email: ticket.email,
      categoria: categoriaLabel(ticket.categoria),
      tipo: tipoLabel(ticket.tipo),
      descricao: ticket.descricao || "",
      data: ticket.created_at ? ticket.created_at.slice(0, 10) : "",
    }], `chamado-${ticket.id.slice(0, 8)}.txt`);
    notify("Chamado exportado.", "success");
  }

  function promptClearHistory() {
    const codeInput = el("input", { class: "input", type: "password", placeholder: "Código administrativo" });

    openModal({
      title: "Limpar histórico de chamados",
      body: el("div", {}, [
        el("p", { style: "margin-bottom:10px", text: "Esta ação apaga TODOS os chamados permanentemente. Informe o código administrativo para confirmar." }),
        codeInput,
      ]),
      primaryLabel: "Limpar",
      danger: true,
      onConfirm: async () => {
        try {
          await API.clearSupportHistory(codeInput.value.trim());
          notify("Histórico de chamados apagado.", "warning");
          load();
        } catch (err) {
          notify(err.message || "Código administrativo inválido.", "error");
        }
      },
    });
  }

  load();
}
