/**
 * Tiny hash-based router. Pages register here in app.js.
 * Navigation never reloads the page.
 */

const routes = new Map();
let guard = null;

export const router = {
  register(path, handler, opts = {}) { routes.set(path, { handler, ...opts }); },
  setGuard(fn) { guard = fn; },
  navigate(path) {
    if (location.hash === "#" + path) { this.resolve(); return; }
    location.hash = path;
  },
  current() { return location.hash.replace(/^#/, "") || "/dashboard"; },

  async resolve() {
    const path = this.current();
    const match = matchRoute(path);
    if (!match) return mount("/not-found", { path });
    if (guard) {
      const decision = guard(match.route, path);
      if (decision && decision.redirect) { this.navigate(decision.redirect); return; }
    }
    mount(match.path, { params: match.params, path });
  },
};

function matchRoute(path) {
  if (routes.has(path)) return { path, route: routes.get(path), params: {} };
  for (const [pattern, route] of routes) {
    if (!pattern.includes(":")) continue;
    const re = new RegExp("^" + pattern.replace(/:[^/]+/g, "([^/]+)") + "$");
    const m = path.match(re);
    if (m) {
      const keys = (pattern.match(/:[^/]+/g) || []).map((k) => k.slice(1));
      const params = Object.fromEntries(keys.map((k, i) => [k, m[i + 1]]));
      return { path: pattern, route, params };
    }
  }
  return null;
}

function mount(path, ctx) {
  const app = document.getElementById("app");
  if (!app) return;
  const route = routes.get(path);
  if (!route) { app.innerHTML = `<div style="padding:40px;text-align:center"><h2>Página não encontrada</h2></div>`; return; }

  // Fade transition on the view container
  const view = document.createElement("div");
  view.className = "route-view";
  view.style.minHeight = "100vh";

  app.innerHTML = "";
  app.appendChild(view);
  route.handler(view, ctx);
}

window.addEventListener("hashchange", () => router.resolve());