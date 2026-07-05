import { el, escapeHtml, renderIcons, debounce } from "../utils/helpers.js";

/**
 * DataTable with search, sort, pagination, sticky header.
 *
 * @param {Object} opts
 * @param {Array} opts.rows
 * @param {Array<{key:string,label:string,render?:(row)=>HTMLElement|string}>} opts.columns
 * @param {number} [opts.pageSize=8]
 * @param {boolean} [opts.search=true]
 */
export function DataTable({ rows, columns, pageSize = 8, search = true }) {
  let state = { rows: rows.slice(), all: rows.slice(), page: 1, sortKey: null, sortDir: 1, query: "" };

  const wrap = el("div", { class: "card" });
  const head = el("div", { class: "card-head" }, [
    el("div", { class: "card-title", text: "Resultados" }),
    search ? el("div", { class: "search-bar", style: "max-width:280px" }, [
      el("i", { "data-lucide": "search" }),
      (() => {
        const inp = el("input", { type: "search", placeholder: "Buscar..." });
        inp.addEventListener("input", debounce((e) => { state.query = e.target.value.toLowerCase(); state.page = 1; recompute(); }, 200));
        return inp;
      })(),
    ]) : el("div"),
  ]);

  const tableWrap = el("div", { class: "table-wrap" });
  const table = el("table", { class: "data" });
  const thead = el("thead"); const tbody = el("tbody");
  table.append(thead, tbody);
  tableWrap.append(table);

  const pagination = el("div", { class: "pagination" });

  wrap.append(head, tableWrap, pagination);

  function renderHeader() {
    thead.innerHTML = "";
    const tr = el("tr");
    for (const col of columns) {
      const isSorted = state.sortKey === col.key;
      const th = el("th", { class: isSorted ? "sorted" : "" }, [
        col.label,
        el("span", { class: "sort-ind", html: isSorted ? (state.sortDir > 0 ? "▲" : "▼") : "↕" }),
      ]);
      th.addEventListener("click", () => {
        if (state.sortKey === col.key) state.sortDir *= -1;
        else { state.sortKey = col.key; state.sortDir = 1; }
        recompute();
      });
      tr.appendChild(th);
    }
    thead.appendChild(tr);
  }

  function recompute() {
    let filtered = state.all;
    if (state.query) {
      filtered = filtered.filter((r) => columns.some((c) => String(r[c.key] ?? "").toLowerCase().includes(state.query)));
    }
    if (state.sortKey) {
      filtered = filtered.slice().sort((a, b) => {
        const av = a[state.sortKey], bv = b[state.sortKey];
        if (av == null) return 1; if (bv == null) return -1;
        if (typeof av === "number" && typeof bv === "number") return (av - bv) * state.sortDir;
        return String(av).localeCompare(String(bv), "pt-BR", { numeric: true }) * state.sortDir;
      });
    }
    state.rows = filtered;
    renderBody();
    renderPagination();
    renderHeader();
  }

  function renderBody() {
    tbody.innerHTML = "";
    const start = (state.page - 1) * pageSize;
    const slice = state.rows.slice(start, start + pageSize);
    if (slice.length === 0) {
      tbody.appendChild(el("tr", {}, [el("td", { colspan: columns.length, class: "muted", style: "text-align:center; padding:30px" }, ["Nenhum resultado encontrado."])]));
      return;
    }
    for (const row of slice) {
      const tr = el("tr");
      for (const col of columns) {
        const td = el("td");
        const v = col.render ? col.render(row) : row[col.key];
        if (v && v.nodeType) td.append(v); else td.innerHTML = escapeHtml(v ?? "");
        tr.appendChild(td);
      }
      tbody.appendChild(tr);
    }
    renderIcons(tbody);
  }

  function renderPagination() {
    pagination.innerHTML = "";
    const total = state.rows.length;
    const pages = Math.max(1, Math.ceil(total / pageSize));
    if (state.page > pages) state.page = pages;
    const info = el("div", { class: "muted", text: `${total} registro(s) · página ${state.page}/${pages}` });
    const btns = el("div", { class: "pager-btns" });
    const mkBtn = (label, page, opts = {}) => {
      const b = el("button", { class: "pager-btn" + (opts.active ? " active" : ""), text: label });
      if (opts.disabled) b.setAttribute("disabled", "");
      b.addEventListener("click", () => { state.page = page; renderBody(); renderPagination(); });
      return b;
    };
    btns.appendChild(mkBtn("‹", Math.max(1, state.page - 1), { disabled: state.page === 1 }));
    const max = Math.min(pages, 5);
    let startP = Math.max(1, state.page - 2);
    if (startP + max - 1 > pages) startP = Math.max(1, pages - max + 1);
    for (let i = 0; i < max; i++) {
      const p = startP + i;
      btns.appendChild(mkBtn(String(p), p, { active: p === state.page }));
    }
    btns.appendChild(mkBtn("›", Math.min(pages, state.page + 1), { disabled: state.page === pages }));
    pagination.append(info, btns);
  }

  function setRows(rows) { state.all = rows.slice(); state.page = 1; recompute(); }

  renderHeader();
  renderBody();
  renderPagination();
  renderIcons(wrap);

  return { node: wrap, setRows, getFiltered: () => state.rows };
}