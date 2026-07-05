import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { notify } from "../components/notifications.js";

export function SettingsPage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    const head = el("div", { class: "page-head" }, [el("div", {}, [el("h1", { text: "Configurações" }), el("p", { class: "muted", text: "Preferências do sistema e segurança." })])]);

    const tabs = ["Geral", "Notificações", "Segurança"];
    let active = "Geral";
    const tabsRow = el("div", { class: "tabs" });
    const body = el("div");
    function renderTabs() {
      tabsRow.innerHTML = "";
      tabs.forEach((t) => {
        const b = el("button", { class: "tab" + (active === t ? " active" : ""), text: t });
        b.addEventListener("click", () => { active = t; renderTabs(); renderBody(); });
        tabsRow.appendChild(b);
      });
    }
    function row(title, desc, control) {
      return el("div", { class: "setting-row" }, [
        el("div", {}, [el("h4", { text: title }), el("p", { text: desc })]),
        control,
      ]);
    }
    function makeSwitch(initial, onChange) {
      const sw = el("div", { class: "switch" + (initial ? " on" : "") });
      sw.addEventListener("click", () => {
        sw.classList.toggle("on");
        onChange?.(sw.classList.contains("on"));
      });
      return sw;
    }
    function renderBody() {
      body.innerHTML = "";
      const card = el("div", { class: "card card-pad" });
      if (active === "Geral") {
        card.append(
          row("Idioma", "Português (Brasil)", el("select", { class: "select", style: "max-width:200px" }, [el("option", { text: "Português (Brasil)" }), el("option", { text: "English" })])),
          row("Tema escuro", "Em breve.", makeSwitch(false, () => notify("Tema será sincronizado em breve.", "info"))),
        );
      } else if (active === "Notificações") {
        card.append(
          row("Alertas por e-mail", "Receba alertas sobre estoque baixo.", makeSwitch(true)),
          row("Push", "Notificações no navegador.", makeSwitch(false)),
          row("Relatórios semanais", "Resumo todas as segundas.", makeSwitch(true)),
        );
      } else {
        card.append(
          row("Autenticação 2 fatores", "Camada extra de proteção.", makeSwitch(false, () => notify("2FA será habilitado após integração.", "info"))),
          row("Sessões ativas", "Desconectar todos os dispositivos.", el("button", { class: "btn btn-outline", text: "Encerrar sessões", onclick: () => notify("Sessões encerradas (simulado).") })),
        );
      }
      body.appendChild(card);
    }
    renderTabs(); renderBody();
    content.append(head, tabsRow, body);
    renderIcons(content);
  });
}
