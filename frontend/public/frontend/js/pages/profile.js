import { el, renderIcons } from "../utils/helpers.js";
import { AppShell } from "./_shell.js";
import { session } from "../services/store.js";
import { API, ROLE_LABEL } from "../services/api.js";
import { notify } from "../components/notifications.js";
import { getInitials } from "../utils/helpers.js";
import { validateEmail, required } from "../utils/validators.js";
import { guardedClick } from "../utils/security.js";

export function ProfilePage(root, ctx) {
  AppShell(root, ctx.path, (content) => {
    let user = session.user || { name: "Usuário", email: "", role: "professor" };

    const avatar = el("div", { class: "avatar avatar-lg", text: getInitials(user.name) });
    const nameDisplay = el("h3", { text: user.name });
    const emailDisplay = el("p", { text: user.email });
    const roleChip = el("span", { class: "chip chip-success", style: "margin-top:10px", text: ROLE_LABEL[user.role] || user.role });
    const linksList = el("div", { class: "sidebar-user-turmas", style: "margin-top:10px" });

    const head = el("div", { class: "page-head" }, [
      el("div", {}, [el("h1", { text: "Perfil" }), el("p", { class: "muted", text: "Suas informações pessoais e vínculos no sistema." })]),
    ]);
    const side = el("div", { class: "card profile-side" }, [avatar, nameDisplay, emailDisplay, roleChip, linksList]);

    const supportCodeCard = user.role === "professor"
      ? el("div", { class: "card card-pad", style: "margin-top:16px" }, [
          el("h3", { text: "Código de suporte", style: "margin-bottom:6px" }),
          el("p", { class: "muted", style: "font-size:0.82em;margin-bottom:10px", text: "Use este código ao abrir um chamado em Suporte." }),
          el("div", { class: "support-code-display", text: user.supportCode || "—" }),
        ])
      : el("span");

    const nameInput = el("input", { class: "input", value: user.name });
    const emailInput = el("input", { class: "input", type: "email", value: user.email });
    const roleInput = el("input", { class: "input", value: ROLE_LABEL[user.role] || user.role, disabled: true });
    const errs = { name: el("div", { class: "error-text" }), email: el("div", { class: "error-text" }) };

    const form = el("form", { class: "card card-pad" }, [
      el("h3", { text: "Informações pessoais", style: "margin-bottom:14px" }),
      el("div", { class: "form-grid" }, [
        el("div", { class: "field" }, [el("label", { text: "Nome" }), nameInput, errs.name]),
        el("div", { class: "field" }, [el("label", { text: "Email" }), emailInput, errs.email]),
        el("div", { class: "field" }, [el("label", { text: "Perfil" }), roleInput]),
      ]),
      el("div", { style: "display:flex;gap:10px;margin-top:18px;justify-content:flex-end" }, [
        el("button", { type: "reset", class: "btn btn-ghost", text: "Cancelar" }),
        el("button", { type: "submit", class: "btn btn-primary", text: "Salvar alterações" }),
      ]),
    ]);

    form.addEventListener("submit", guardedClick((e) => {
      e.preventDefault();
      const nE = required(nameInput.value, "Nome"); errs.name.textContent = nE || "";
      const eE = validateEmail(emailInput.value); errs.email.textContent = eE || "";
      if (nE || eE) return;
      avatar.textContent = getInitials(nameInput.value);
      nameDisplay.textContent = nameInput.value;
      emailDisplay.textContent = emailInput.value;
      notify("A edição de nome/e-mail ainda não é suportada pelo backend — nenhum endpoint de atualização de perfil foi especificado.", "info");
    }));

    content.append(head, el("div", { class: "profile-grid" }, [el("div", {}, [side, supportCodeCard]), form]));
    renderIcons(content);

    function renderLinks(freshUser) {
      linksList.innerHTML = "";
      if (freshUser.role === "professor" && freshUser.classes?.length) {
        linksList.textContent = "Turmas: " + freshUser.classes.map((c) => c.name).join(", ");
      } else if (freshUser.deposits?.length) {
        linksList.textContent = "Depósitos: " + freshUser.deposits.map((d) => d.name).join(", ");
      }
    }
    renderLinks(user);

    // Atualiza com dados reais do backend (turmas/depósitos vinculados).
    API.me().then((freshUser) => {
      user = freshUser;
      nameInput.value = freshUser.name;
      emailInput.value = freshUser.email;
      roleInput.value = ROLE_LABEL[freshUser.role] || freshUser.role;
      nameDisplay.textContent = freshUser.name;
      emailDisplay.textContent = freshUser.email;
      roleChip.textContent = ROLE_LABEL[freshUser.role] || freshUser.role;
      avatar.textContent = getInitials(freshUser.name);
      renderLinks(freshUser);
      if (freshUser.role === "professor") {
        const codeEl = supportCodeCard.querySelector?.(".support-code-display");
        if (codeEl) codeEl.textContent = freshUser.supportCode || "—";
      }
      session.signIn(freshUser, session.token);
    }).catch(() => {
      // Mantém os dados da sessão local se a chamada falhar.
    });
  });
}
