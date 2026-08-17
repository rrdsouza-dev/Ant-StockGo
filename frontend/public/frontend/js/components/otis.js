/**
 * otis.js — Otis, o assistente de IA do ANT-Stock.
 *
 * Padrão de montagem: igual a notifications.js — um único componente
 * global, montado uma vez em document.body fora do fluxo do router
 * (que remonta #app inteiro a cada navegação, ver router.js). Otis
 * observa a sessão diretamente e aparece/some sozinho no login/logout,
 * sem que nenhuma página precise saber que ele existe.
 *
 * Conversa: mantida apenas em memória neste módulo enquanto o usuário
 * usa o sistema (ver "CHAT TEMPORÁRIO" na especificação). Nada é
 * persistido em localStorage/sessionStorage nem em banco nesta V1.
 */
import { el, renderIcons, escapeHtml } from "../utils/helpers.js";
import { session } from "../services/store.js";
import { API } from "../services/api.js";
import { guardedClick } from "../utils/security.js";

const SUGGESTED_QUESTIONS = [
  "Como cadastro um produto?",
  "Como registro uma entrada de estoque?",
  "Qual a diferença entre depósito e turma?",
];

let messages = []; // { role: "user" | "assistant", content: string }
let isOpen = false;
let isSending = false;

let root, bubble, panel, messagesEl, form, input, sendBtn;

function initialGreetingNode() {
  return el("div", { class: "otis-empty" }, [
    el("div", { class: "otis-empty-icon" }, [starIcon(22)]),
    el("p", { class: "otis-empty-title", text: "Olá! Eu sou o Otis, assistente do ANT-Stock." }),
    el("p", { class: "otis-empty-text", text: "Posso ajudar você a entender e utilizar o sistema — cadastro de produtos, movimentações, depósitos, turmas e relatórios." }),
    el("div", { class: "otis-suggestions" }, SUGGESTED_QUESTIONS.map((q) =>
      el("button", { type: "button", class: "otis-suggestion", text: q, onClick: () => sendMessage(q) })
    )),
  ]);
}

/** Ícone de estrela do Otis — desenhado à mão (sem depender de um ícone Lucide específico), com traço arredondado para casar com o strokeWidth 1.8 usado no resto do sistema. */
function starIcon(size = 22) {
  const ns = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(ns, "svg");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("width", String(size));
  svg.setAttribute("height", String(size));
  svg.setAttribute("fill", "none");
  svg.classList.add("otis-star");
  const path = document.createElementNS(ns, "path");
  path.setAttribute(
    "d",
    "M12 2.5c.35 0 .66.23.76.57l1.53 5.12a3.6 3.6 0 0 0 2.42 2.42l5.12 1.53a.8.8 0 0 1 0 1.53l-5.12 1.53a3.6 3.6 0 0 0-2.42 2.42l-1.53 5.12a.8.8 0 0 1-1.53 0l-1.53-5.12a3.6 3.6 0 0 0-2.42-2.42L2.16 13.7a.8.8 0 0 1 0-1.53l5.12-1.53a3.6 3.6 0 0 0 2.42-2.42l1.53-5.12A.79.79 0 0 1 12 2.5Z"
  );
  path.setAttribute("fill", "currentColor");
  svg.appendChild(path);
  return svg;
}

function messageNode({ role, content }) {
  return el("div", { class: `otis-msg otis-msg-${role}` }, [
    el("div", { class: "otis-bubble", html: formatContent(content) }),
  ]);
}

/** Escapa o conteúdo e converte quebras de linha em <br>; nada além disso — sem markdown completo nesta V1. */
function formatContent(text) {
  return escapeHtml(text).replace(/\n/g, "<br>");
}

function typingNode() {
  return el("div", { class: "otis-msg otis-msg-assistant", "data-typing": "1" }, [
    el("div", { class: "otis-bubble otis-typing" }, [
      el("span", { class: "otis-dot" }),
      el("span", { class: "otis-dot" }),
      el("span", { class: "otis-dot" }),
    ]),
  ]);
}

function renderMessages() {
  messagesEl.innerHTML = "";
  if (messages.length === 0) {
    messagesEl.appendChild(initialGreetingNode());
  } else {
    for (const m of messages) messagesEl.appendChild(messageNode(m));
  }
  renderIcons(messagesEl);
  scrollToBottom();
}

function scrollToBottom() {
  requestAnimationFrame(() => { messagesEl.scrollTop = messagesEl.scrollHeight; });
}

function setSending(sending) {
  isSending = sending;
  sendBtn.disabled = sending || !input.value.trim();
  input.disabled = sending;
  bubble.classList.toggle("otis-bubble-btn-processing", sending);
}

async function sendMessage(text) {
  const message = (text ?? input.value).trim();
  if (!message || isSending) return;

  input.value = "";
  autoGrow();
  messages.push({ role: "user", content: message });
  renderMessages();

  setSending(true);
  const typing = typingNode();
  messagesEl.appendChild(typing);
  scrollToBottom();

  try {
    // Envia só os últimos turnos como histórico — suficiente para dar
    // continuidade à conversa sem depender de persistência no backend.
    const history = messages.slice(0, -1).slice(-20);
    const response = await API.otisChat(message, history);
    typing.remove();
    messages.push({ role: "assistant", content: response });
    renderMessages();
  } catch (err) {
    typing.remove();
    messagesEl.appendChild(
      el("div", { class: "otis-error" }, [
        el("i", { "data-lucide": "alert-triangle" }),
        el("span", { text: err.message || "Não foi possível falar com o Otis agora." }),
      ])
    );
    renderIcons(messagesEl);
    scrollToBottom();
  } finally {
    setSending(false);
  }
}

function clearConversation() {
  messages = [];
  renderMessages();
}

function autoGrow() {
  input.style.height = "auto";
  input.style.height = Math.min(input.scrollHeight, 120) + "px";
}

function openPanel() {
  if (isOpen) return;
  isOpen = true;
  panel.classList.add("otis-panel-open");
  bubble.classList.add("otis-bubble-btn-active");
  bubble.setAttribute("aria-expanded", "true");
  scrollToBottom();
  setTimeout(() => input.focus(), 180);
}

function closePanel() {
  if (!isOpen) return;
  isOpen = false;
  panel.classList.remove("otis-panel-open");
  bubble.classList.remove("otis-bubble-btn-active");
  bubble.setAttribute("aria-expanded", "false");
}

function togglePanel() {
  if (isOpen) closePanel();
  else openPanel();
}

function buildPanel() {
  const title = el("div", { class: "otis-panel-title" }, [
    el("span", { class: "otis-title-icon" }, [starIcon(16)]),
    el("span", { text: "Otis" }),
  ]);

  const clearBtn = el("button", {
    type: "button",
    class: "otis-icon-btn",
    title: "Limpar conversa",
    "aria-label": "Limpar conversa",
    onClick: clearConversation,
  }, [el("i", { "data-lucide": "eraser" })]);

  const closeBtn = el("button", {
    type: "button",
    class: "otis-icon-btn",
    title: "Fechar",
    "aria-label": "Fechar",
    onClick: closePanel,
  }, [el("i", { "data-lucide": "x" })]);

  const header = el("div", { class: "otis-panel-header" }, [title, el("div", { class: "otis-header-actions" }, [clearBtn, closeBtn])]);

  messagesEl = el("div", { class: "otis-messages" });

  input = el("textarea", {
    class: "otis-input",
    rows: "1",
    placeholder: "Digite uma mensagem...",
    onInput: () => { autoGrow(); sendBtn.disabled = isSending || !input.value.trim(); },
    onKeydown: (e) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        if (!sendBtn.disabled) form.requestSubmit();
      }
    },
  });

  sendBtn = el("button", { type: "submit", class: "otis-send-btn", disabled: true, "aria-label": "Enviar" }, [
    el("i", { "data-lucide": "arrow-up" }),
  ]);

  form = el("form", { class: "otis-input-row" }, [input, sendBtn]);
  // Único ponto de envio (clique no botão ou Enter, via requestSubmit
  // acima). isSending bloqueia reentrância durante a requisição;
  // guardedClick some como segunda camada contra cliques/Enters muito
  // rápidos em sequência antes da UI atualizar.
  form.addEventListener("submit", guardedClick((e) => { e.preventDefault(); return sendMessage(); }, { cooldown: 300 }));

  panel = el("div", { class: "otis-panel", role: "dialog", "aria-label": "Otis, assistente do ANT-Stock" }, [
    header,
    messagesEl,
    form,
  ]);

  return panel;
}

function buildBubble() {
  bubble = el("button", {
    type: "button",
    class: "otis-bubble-btn",
    "aria-label": "Abrir o Otis, assistente do ANT-Stock",
    "aria-expanded": "false",
    onClick: togglePanel,
  }, [starIcon(24), el("span", { class: "otis-spinner-ring" })]);
  return bubble;
}

function mount() {
  if (root) return; // já montado
  root = el("div", { class: "otis-root" }, [buildBubble(), buildPanel()]);
  document.body.appendChild(root);
  renderMessages();
  renderIcons(root);
}

// Registrado uma única vez (fora de mount/unmount) para não acumular
// listeners a cada ciclo de login/logout.
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && isOpen) closePanel();
});

function unmount() {
  if (!root) return;
  root.remove();
  root = null;
  isOpen = false;
  messages = [];
}

function syncVisibility() {
  if (session.isAuthenticated()) mount();
  else unmount();
}

session.subscribe(syncVisibility);
syncVisibility();
