import { el, renderIcons, getInitials } from "../utils/helpers.js";
import { session } from "../services/store.js";
import { router } from "../router.js";
import { getNotificationLog, subscribeNotificationLog } from "./notifications.js";
import { debounce } from "../utils/helpers.js";

export function Navbar({ onSearch } = {}) {
  const user = session.user || { name: "Convidado", role: "Usuário" };
  const search = el("div", { class: "search-bar" }, [
    el("i", { "data-lucide": "search" }),
    el("input", { type: "text", placeholder: "Buscar itens de estoque ou professores" }),
  ]);
  search.querySelector("input").addEventListener("input", debounce((e) => onSearch?.(e.target.value), 220));

  const bellWrap = NotificationBell();

  const userPill = el("div", { class: "user-pill" }, [
    el("div", { class: "avatar", text: getInitials(user.name) }),
    el("div", { class: "name", text: user.name }),
  ]);
  const logout = el("button", { class: "icon-btn", title: "Sair", onclick: () => { session.signOut(); router.navigate("/login"); } }, [
    el("i", { "data-lucide": "log-out" }),
  ]);

  const bar = el("header", { class: "topbar" }, [
    search,
    el("div", { class: "right" }, [bellWrap, userPill, logout]),
  ]);

  renderIcons(bar);
  return bar;
}

/**
 * NotificationBell — sino da navbar. Ao clicar, abre um pequeno card
 * listando as notificações da sessão atual (ver components/notifications.js
 * para a origem dos dados — tudo em memória, sem banco/API/persistência).
 *
 * O ponto azul só aparece quando há ao menos uma notificação registrada,
 * e some sozinho quando o histórico é zerado (logout).
 */
function NotificationBell() {
  const dot = el("span", { class: "dot" });
  const bellBtn = el("button", { class: "icon-btn", title: "Notificações" }, [
    el("i", { "data-lucide": "bell" }), dot,
  ]);
  const list = el("div", { class: "notif-dropdown-list" });
  const dropdown = el("div", { class: "notif-dropdown" }, [
    el("div", { class: "notif-dropdown-head" }, ["Notificações"]),
    list,
  ]);
  const wrap = el("div", { class: "notif-bell-wrap" }, [bellBtn, dropdown]);

  let isOpen = false;

  function renderList() {
    const log = getNotificationLog();
    dot.style.display = log.length ? "block" : "none";
    list.innerHTML = "";
    if (!log.length) {
      list.appendChild(el("div", { class: "notif-dropdown-empty" }, ["Nenhuma notificação nesta sessão."]));
      return;
    }
    log.forEach((n) => {
      list.appendChild(el("div", { class: "notif-dropdown-item" }, [
        el("span", { class: `notif-dot notif-dot-${n.type}` }),
        el("div", { class: "notif-dropdown-text" }, [
          el("div", { class: "notif-dropdown-msg", text: n.message }),
          el("div", { class: "notif-dropdown-time", text: formatTime(n.time) }),
        ]),
      ]));
    });
  }

  function onOutsideClick(e) {
    if (!wrap.contains(e.target)) closeDropdown();
  }

  function openDropdown() {
    isOpen = true;
    dropdown.classList.add("open");
    renderList();
    // Listener adicionado só enquanto o dropdown está aberto, e removido
    // ao fechar — evita acumular listeners em `document` a cada navegação
    // (a navbar é recriada do zero em toda troca de rota).
    setTimeout(() => document.addEventListener("click", onOutsideClick), 0);
  }

  function closeDropdown() {
    isOpen = false;
    dropdown.classList.remove("open");
    document.removeEventListener("click", onOutsideClick);
  }

  bellBtn.addEventListener("click", (e) => {
    e.stopPropagation();
    if (isOpen) closeDropdown(); else openDropdown();
  });

  subscribeNotificationLog(() => {
    dot.style.display = getNotificationLog().length ? "block" : "none";
    if (isOpen) renderList();
  });

  renderList(); // define o estado inicial do ponto, mesmo com o card fechado

  return wrap;
}

function formatTime(date) {
  const d = date instanceof Date ? date : new Date(date);
  return d.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });
}
