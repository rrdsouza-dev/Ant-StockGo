import { el, renderIcons } from "../utils/helpers.js";
import { Sidebar } from "../components/sidebar.js";
import { Navbar } from "../components/navbar.js";

/** Layout shell shared by all authenticated pages. */
export function AppShell(root, currentPath, contentBuilder, { onSearch } = {}) {
  document.body.classList.add("app-bg");
  const sidebar = Sidebar(currentPath);
  const navbar = Navbar({ onSearch });
  const content = el("main", { class: "content stagger" });
  const main = el("section", { class: "main" }, [navbar, content]);
  const shell = el("div", { class: "app-shell" }, [sidebar, main]);
  root.appendChild(shell);
  contentBuilder(content);
  renderIcons(content);
}