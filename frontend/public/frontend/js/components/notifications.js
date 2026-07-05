import { el, renderIcons } from "../utils/helpers.js";

const ICONS = { success: "check-circle-2", error: "alert-triangle", warning: "alert-circle", info: "info" };

export function notify(message, type = "success", { duration = 3200 } = {}) {
  const container = document.getElementById("notifications");
  if (!container) return;
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