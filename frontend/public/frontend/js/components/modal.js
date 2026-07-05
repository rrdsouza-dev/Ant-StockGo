import { el, renderIcons } from "../utils/helpers.js";

export function openModal({ title, body, primaryLabel = "Confirmar", onConfirm, danger = false }) {
  const cancel = el("button", { class: "btn btn-ghost", text: "Cancelar" });
  const confirm = el("button", { class: `btn ${danger ? "btn-danger" : "btn-primary"}`, text: primaryLabel });
  const card = el("div", { class: "modal" }, [
    el("h3", { text: title }),
    typeof body === "string" ? el("p", { class: "muted", text: body }) : body,
    el("div", { class: "modal-actions" }, [cancel, confirm]),
  ]);
  const backdrop = el("div", { class: "modal-backdrop" }, [card]);
  const close = () => backdrop.remove();
  backdrop.addEventListener("click", (e) => { if (e.target === backdrop) close(); });
  cancel.addEventListener("click", close);
  confirm.addEventListener("click", async () => { try { await onConfirm?.(); } finally { close(); } });
  document.body.appendChild(backdrop);
  renderIcons(backdrop);
  return close;
}