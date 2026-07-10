/**
 * choose-class.js — "Qual turma deseja acessar?"
 *
 * Mostrada ao professor logo após o login quando ele está vinculado a
 * mais de uma turma (se só houver uma, o app entra direto nela — ver
 * app.js). Também acessível a qualquer momento pelo item "Trocar turma"
 * na sidebar, permitindo mudar de turma sem precisar sair e entrar de
 * novo. A turma escolhida fica em session.classId e é usada pelo
 * backend (via ?class_id=) para restringir o estoque visível a essa
 * turma especificamente — ver InventoryService/DepositService.
 */
import { el, renderIcons } from "../utils/helpers.js";
import { router } from "../router.js";
import { API } from "../services/api.js";
import { session } from "../services/store.js";
import { notify } from "../components/notifications.js";
import { guardedClick } from "../utils/security.js";

export function ChooseClassPage(root) {
  document.body.classList.add("app-bg");

  const list = el("div", { class: "choose-class-list" }, [
    el("div", { class: "muted", style: "padding:30px;text-align:center" }, ["Carregando suas turmas…"]),
  ]);

  const card = el("div", { class: "auth-card choose-class-card" }, [
    el("div", { class: "brand" }, [el("img", { src: "assets/images/logo-light.jpg", alt: "ANT Stock" })]),
    el("h2", { text: "Qual turma deseja acessar?" }),
    el("p", { class: "subtitle", text: "Você está vinculado a mais de uma turma. Escolha com qual deseja trabalhar agora — você pode trocar depois pela barra lateral." }),
    list,
  ]);

  root.appendChild(el("div", { class: "auth-center" }, [card]));
  renderIcons(root);

  async function load() {
    try {
      const classes = await API.classes();
      list.innerHTML = "";
      if (!classes.length) {
        list.appendChild(el("p", { class: "muted", text: "Você ainda não está vinculado a nenhuma turma. Fale com a gestão." }));
        return;
      }
      classes.forEach((cls) => {
        const depositNames = (cls.deposits || []).map((d) => d.name).join(", ") || "Nenhum depósito vinculado";
        const btn = el("button", { type: "button", class: "choose-class-item", onclick: guardedClick(() => choose(cls)) }, [
          el("div", {}, [
            el("div", { class: "choose-class-name", text: cls.name }),
            el("div", { class: "choose-class-deposits", text: depositNames }),
          ]),
          el("i", { "data-lucide": "chevron-right" }),
        ]);
        list.appendChild(btn);
      });
      renderIcons(list);
    } catch (err) {
      list.innerHTML = "";
      list.appendChild(el("p", { class: "muted", text: "Erro ao carregar turmas." }));
      notify(err.message || "Erro ao carregar turmas.", "error");
    }
  }

  function choose(cls) {
    session.setClassId(cls.id);
    notify(`Turma "${cls.name}" selecionada.`, "success");
    router.navigate("/dashboard");
  }

  load();
}
