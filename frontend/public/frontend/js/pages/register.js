import { el, renderIcons } from "../utils/helpers.js";
import { router } from "../router.js";
import { API } from "../services/api.js";
import { notify } from "../components/notifications.js";
import { validateEmail, validatePassword, passwordScore, required } from "../utils/validators.js";
import { guardedClick } from "../utils/security.js";

export function RegisterPage(root) {
  document.body.classList.add("app-bg");

  const form = el("form", { class: "auth-form", novalidate: true });
  const errs = {
    name: el("div", { class: "error-text" }),
    email: el("div", { class: "error-text" }),
    pass: el("div", { class: "error-text" }),
    conf: el("div", { class: "error-text" }),
  };
  const strength = el("div", { class: "strength" }, [el("span"), el("span"), el("span"), el("span")]);

  const name  = el("input", { class: "input", name: "name", placeholder: "Nome completo" });
  const email = el("input", { class: "input", type: "email", name: "email", placeholder: "Email" });
  const pass  = el("input", { class: "input", type: "password", name: "password", placeholder: "Senha (mín. 8 caracteres, 1 maiúscula, 1 número, 1 símbolo)" });
  const conf  = el("input", { class: "input", type: "password", name: "confirm", placeholder: "Confirmar senha" });

  // ── Perfil ──────────────────────────────────────────────────
  const perfilGroup = el("div", { class: "perfil-selector" });

  const perfis = [
    { value: "professor", label: "Professor" },
    { value: "gestao",    label: "Gestão" },
  ];

  let perfilSelecionado = "professor";

  perfis.forEach(({ value, label }) => {
    const btn = el("button", {
      type: "button",
      class: "btn-perfil" + (value === perfilSelecionado ? " active" : ""),
      "data-value": value,
      text: label,
    });
    btn.addEventListener("click", () => {
      perfilSelecionado = value;
      perfilGroup.querySelectorAll(".btn-perfil").forEach(b => b.classList.remove("active"));
      btn.classList.add("active");
    });
    perfilGroup.appendChild(btn);
  });

  pass.addEventListener("input", () => {
    const s = passwordScore(pass.value);
    strength.className = "strength s" + s;
  });

  const submit = el("button", { class: "btn btn-primary btn-lg btn-block", text: "Criar conta" });

  form.append(
    el("div", { class: "field" }, [name, errs.name]),
    el("div", { class: "field" }, [email, errs.email]),
    el("div", { class: "field" }, [pass, strength, errs.pass]),
    el("div", { class: "field" }, [conf, errs.conf]),
    el("div", { class: "field" }, [
      el("label", { class: "field-label", text: "Perfil" }),
      perfilGroup,
      el("p", { class: "muted", style: "font-size:0.8em;margin-top:6px", text: "As turmas e depósitos são vinculados pela gestão após a aprovação da sua conta." }),
    ]),
    submit,
  );

  const card = el("div", { class: "auth-card" }, [
    el("div", { class: "brand" }, [el("img", { src: "assets/images/logo-light.jpg", alt: "ANT Stock" })]),
    el("h2", { text: "Crie sua conta" }),
    el("p", { class: "subtitle", text: "Seu cadastro será revisado pela gestão antes da ativação." }),
    form,
    el("a", { class: "back-link", href: "#/login" }, [el("i", { "data-lucide": "arrow-left" }), " Voltar para o login"]),
  ]);

  form.addEventListener("submit", guardedClick(async (e) => {
    e.preventDefault();
    const nErr = required(name.value, "Nome"); errs.name.textContent = nErr || "";
    const eErr = validateEmail(email.value); errs.email.textContent = eErr || "";
    const pErr = validatePassword(pass.value); errs.pass.textContent = pErr || "";
    const cErr = pass.value !== conf.value ? "As senhas não conferem." : null;
    errs.conf.textContent = cErr || "";
    if (nErr || eErr || pErr || cErr) return;

    submit.innerHTML = ""; submit.appendChild(el("span", { class: "spinner" }));
    try {
      await API.register({
        name: name.value.trim(),
        email: email.value.trim(),
        password: pass.value,
        role: perfilSelecionado,
      });

      // Conta criada como PENDENTE — nunca autentica automaticamente.
      card.innerHTML = "";
      card.append(
        el("div", { class: "brand" }, [el("img", { src: "assets/images/logo-light.jpg", alt: "ANT Stock" })]),
        el("div", { class: "icon-pill", style: "margin:0 auto 16px" }, [el("i", { "data-lucide": "clock" })]),
        el("h2", { text: "Cadastro enviado!" }),
        el("p", { class: "subtitle", text: "Sua conta foi enviada para aprovação da gestão. Você poderá entrar assim que ela for ativada." }),
        el("a", { class: "btn btn-primary btn-lg btn-block", href: "#/login", text: "Voltar para o login" }),
      );
      renderIcons(card);
      notify("Cadastro enviado! Aguarde a aprovação da gestão.", "success");
    } catch (err) {
      notify(err?.message || "Não foi possível criar a conta.", "error");
      submit.textContent = "Criar conta";
    }
  }));

  root.appendChild(el("div", { class: "auth-center" }, [card]));
  renderIcons(root);
}
