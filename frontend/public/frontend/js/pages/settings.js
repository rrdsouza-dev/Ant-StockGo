/**
 * Configurações — hoje contém apenas o período de retenção de
 * movimentações (item novo: a Gestão pode definir depois de quantos dias
 * o histórico de entradas/saídas é apagado automaticamente pelo
 * backend). Esta tela substitui a antiga aba "Configurações" (removida
 * numa versão anterior por ser inteiramente decorativa) — é uma página
 * nova, propositalmente enxuta, criada só para esta funcionalidade.
 */
import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { notify } from "../components/notifications.js";
import { guardedClick } from "../utils/security.js";

const PRESETS = [30, 40, 60];

export function SettingsPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    const isGestao = session.user?.role === "gestao";

    const head = el("div", { class: "page-head" }, [
      el("div", {}, [
        el("h1", { text: "Configurações" }),
        el("p", { class: "muted", text: "Ajustes gerais do sistema." }),
      ]),
    ]);
    content.append(head);
    renderIcons(content);

    const card = el("div", { class: "card card-pad" }, [
      el("h3", { text: "Retenção de movimentações", style: "margin-bottom:6px" }),
      el("p", { class: "muted", style: "font-size:0.85em;margin-bottom:16px", text:
        "Movimentações de entrada/saída mais antigas que o período abaixo são apagadas automaticamente pelo sistema, uma vez por hora. Esta ação é permanente." }),
      el("div", { class: "muted", style: "padding:20px;text-align:center", text: "Carregando…" }),
    ]);
    content.appendChild(card);

    async function load() {
      let current;
      try {
        current = await API.getRetentionSettings();
      } catch (err) {
        card.lastElementChild.replaceWith(el("div", { class: "muted", style: "padding:20px;text-align:center", text: "Erro ao carregar configurações." }));
        notify(err.message || "Erro.", "error");
        return;
      }

      const presetRow = el("div", { class: "perfil-selector" });
      let selectedDays = current.movement_retention_days;

      const customInput = el("input", { type: "number", class: "input", style: "max-width:140px", min: "7", max: "365", value: selectedDays });

      PRESETS.forEach((days) => {
        const btn = el("button", {
          type: "button",
          class: "btn-perfil" + (selectedDays === days ? " active" : ""),
          text: `${days} dias`,
        });
        btn.addEventListener("click", () => {
          selectedDays = days;
          customInput.value = days;
          presetRow.querySelectorAll(".btn-perfil").forEach((b) => b.classList.remove("active"));
          btn.classList.add("active");
        });
        presetRow.appendChild(btn);
      });
      customInput.addEventListener("input", () => {
        presetRow.querySelectorAll(".btn-perfil").forEach((b) => b.classList.remove("active"));
      });

      const errEl = el("div", { class: "error-text" });
      const saveBtn = el("button", { class: "btn btn-primary", text: "Salvar" });
      saveBtn.disabled = !isGestao;

      const body = el("div", {}, [
        el("div", { class: "field" }, [el("label", { class: "field-label", text: "Período (dias)" }), presetRow]),
        el("div", { class: "field", style: "max-width:200px" }, [el("label", { class: "field-label", text: "Ou um valor personalizado (7 a 365)" }), customInput]),
        errEl,
        el("div", { style: "display:flex;justify-content:flex-end;margin-top:10px" }, [saveBtn]),
        !isGestao ? el("p", { class: "muted", style: "font-size:0.8em;margin-top:8px", text: "Somente a gestão pode alterar este período." }) : el("span"),
      ]);

      card.lastElementChild.replaceWith(body);
      renderIcons(card);

      saveBtn.addEventListener("click", guardedClick(async () => {
        const days = Number(customInput.value);
        errEl.textContent = "";
        if (!Number.isInteger(days) || days < 7 || days > 365) {
          errEl.textContent = "Informe um valor entre 7 e 365 dias.";
          return;
        }
        saveBtn.disabled = true;
        try {
          await API.updateRetentionSettings(days);
          notify("Período de retenção atualizado!", "success");
        } catch (err) {
          errEl.textContent = err.message || "Erro ao salvar.";
        } finally {
          saveBtn.disabled = false;
        }
      }));
    }

    load();
  });
}
