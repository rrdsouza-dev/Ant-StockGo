import { el, renderIcons } from "../utils/helpers.js";
import { session } from "../services/store.js";

const ICONS = { success: "check-circle-2", error: "alert-triangle", warning: "alert-circle", info: "info" };

// ── Histórico de notificações da sessão (sino da navbar) ───────────────
// Vive inteiramente em memória no frontend — sem tabela, sem migration,
// sem endpoint, sem localStorage/sessionStorage. Zerado automaticamente
// sempre que a sessão termina, não importa de onde o logout foi
// disparado (navbar, sidebar, ou expiração de token em api.js), porque
// este módulo observa diretamente o estado da sessão.
const sessionLog = [];
const logListeners = new Set();

function emitLog() {
  for (const fn of logListeners) fn(sessionLog);
}

export function subscribeNotificationLog(fn) {
  logListeners.add(fn);
  return () => logListeners.delete(fn);
}

export function getNotificationLog() {
  return sessionLog;
}

let lastUserId = session.user?.id ?? null;
session.subscribe((state) => {
  const currentUserId = state.user?.id ?? null;
  if (currentUserId !== lastUserId) {
    lastUserId = currentUserId;
    if (sessionLog.length) {
      sessionLog.length = 0;
      emitLog();
    }
  }
});

/**
 * notify — toast temporário (sempre exibido) + registro no sino (apenas
 * quando a ação representa uma operação concluída com sucesso).
 *
 * Por padrão, `record` segue o `type`: "success" e "warning" contam como
 * operação concluída (neste projeto "warning" é usado para ações
 * destrutivas bem-sucedidas, como excluir um depósito ou rejeitar uma
 * conta — a operação terminou com sucesso, só o estilo do toast chama
 * mais atenção). "error" nunca é registrado. Para avisos que não
 * representam uma ação concluída (ex.: "selecione um depósito"), passe
 * explicitamente `{ record: false }`.
 */
export function notify(message, type = "success", { duration = 3200, record } = {}) {
  const container = document.getElementById("notifications");
  if (container) {
    const node = el("div", { class: `notification ${type}` }, [
      el("i", { "data-lucide": ICONS[type] || ICONS.success }),
      el("div", { class: "msg", text: message }),
    ]);
    container.appendChild(node);
    renderIcons(node);
    setTimeout(() => {
      node.style.transition = "opacity .25s ease, transform .25s ease";
      node.style.opacity = "0"; node.style.transform = "translateX(12px)";
      setTimeout(() => node.remove(), 260);
    }, duration);
  }

  const shouldRecord = record ?? (type === "success" || type === "warning");
  if (shouldRecord) {
    sessionLog.unshift({ id: `${Date.now()}-${Math.random()}`, message, type, time: new Date() });
    if (sessionLog.length > 30) sessionLog.length = 30; // limite defensivo, sem persistência
    emitLog();
  }
}