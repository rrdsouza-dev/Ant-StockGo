import { el, renderIcons, getInitials } from "../utils/helpers.js";
import { session } from "../services/store.js";
import { router } from "../router.js";
import { notify } from "./notifications.js";
import { debounce } from "../utils/helpers.js";

export function Navbar({ onSearch } = {}) {
  const user = session.user || { name: "Convidado", role: "Usuário" };
  const search = el("div", { class: "search-bar" }, [
    el("i", { "data-lucide": "search" }),
    el("input", { type: "text", placeholder: "Buscar itens de estoque ou professores" }),
  ]);
  search.querySelector("input").addEventListener("input", debounce((e) => onSearch?.(e.target.value), 220));

  const bell = el("button", { class: "icon-btn", title: "Notificações", onclick: () => notify("Sem notificações novas.", "info") }, [
    el("i", { "data-lucide": "bell" }), el("span", { class: "dot" }),
  ]);
  const userPill = el("div", { class: "user-pill" }, [
    el("div", { class: "avatar", text: getInitials(user.name) }),
    el("div", { class: "name", text: user.name }),
  ]);
  const logout = el("button", { class: "icon-btn", title: "Sair", onclick: () => { session.signOut(); router.navigate("/login"); } }, [
    el("i", { "data-lucide": "log-out" }),
  ]);

  const bar = el("header", { class: "topbar" }, [
    search,
    el("div", { class: "right" }, [bell, userPill, logout]),
  ]);

  renderIcons(bar);
  return bar;
}
