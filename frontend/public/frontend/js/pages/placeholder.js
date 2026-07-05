import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";

/** Generic page used for sections that aren't fully spec'd yet but must
 * exist in the menu (ex.: Suporte). */
export function PlaceholderPage(title, description, icon = "construction") {
  return function (root, ctx) {
    AppShell(root, ctx.path, (content) => {
      content.append(
        el("div", { class: "page-head" }, [el("div", {}, [el("h1", { text: title }), el("p", { class: "muted", text: description })])]),
        el("div", { class: "card card-pad", style: "text-align:center;padding:60px 20px" }, [
          el("div", { style: "width:64px;height:64px;border-radius:50%;background:var(--green-50);color:var(--green-700);display:inline-flex;align-items:center;justify-content:center;margin-bottom:14px" }, [el("i", { "data-lucide": icon, style: "width:30px;height:30px" })]),
          el("h3", { text: "Em breve" }),
          el("p", { class: "muted", text: "Esta área será conectada ao backend Go/Supabase nas próximas etapas." }),
        ]),
      );
      renderIcons(content);
    });
  };
}