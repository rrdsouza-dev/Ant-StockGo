/**
 * Leitor de código de barras USB (modo teclado).
 *
 * Leitores USB se comportam como teclado: digitam o código e pressionam Enter.
 * Implementamos:
 *  - Campo com foco automático
 *  - Captura via "keydown" global (modo background) ou campo visível
 *  - Callback com o código lido
 *  - Histórico de leituras temporário (apenas no navegador, não afeta o banco)
 */

import { el, renderIcons } from "../utils/helpers.js";

// Histórico de leituras é puramente local/temporário (não é uma movimentação real).
// "Limpar Leituras" apaga somente este histórico, nunca os dados do backend.
const BarcodeHistory = {
  _key: "antstock:localdb:barcodeHistory",
  list() {
    try {
      return Object.values(JSON.parse(localStorage.getItem(this._key) || "{}"));
    } catch {
      return [];
    }
  },
  save(entry) {
    try {
      const table = JSON.parse(localStorage.getItem(this._key) || "{}");
      table[entry.id] = entry;
      const keys = Object.keys(table).sort();
      if (keys.length > 20) delete table[keys[0]];
      localStorage.setItem(this._key, JSON.stringify(table));
    } catch {
      /* silent */
    }
  },
  clear() {
    try {
      localStorage.removeItem(this._key);
    } catch {
      /* silent */
    }
  },
};

const SCANNER_MIN_LEN = 3; // tamanho mínimo para considerar código válido

/**
 * Cria um widget de leitura de código de barras.
 *
 * @param {function} onScan - callback({ code }) chamado quando código é lido
 * @param {object}   opts
 *   opts.autoFocus  {boolean} foca o campo ao montar (default true)
 *   opts.background {boolean} captura globalmente sem campo visível (default false)
 *   opts.label      {string}  texto do campo
 */
export function BarcodeScanner({
  onScan,
  autoFocus = true,
  background = false,
  label = "Código de barras / QR",
} = {}) {
  let lastKeyTime = 0;

  const input = el("input", {
    class: "input barcode-input",
    placeholder: "Escaneie ou digite o código…",
    autocomplete: "off",
    spellcheck: "false",
  });

  const statusEl = el("div", {
    class: "barcode-status muted",
    text: "Aguardando leitura…",
  });
  const historyList = el("ul", { class: "barcode-history-list" });

  const clearHistoryBtn = el(
    "button",
    { type: "button", class: "btn btn-ghost btn-sm", title: "Limpar leituras temporárias (não apaga movimentações)" },
    [el("i", { "data-lucide": "eraser" }), " Limpar Leituras"],
  );
  clearHistoryBtn.addEventListener("click", () => {
    BarcodeHistory.clear();
    refreshHistory();
  });

  const wrapper = el("div", { class: "barcode-scanner-widget" }, [
    el("label", { class: "field-label", text: label }),
    el("div", { class: "barcode-row" }, [
      el("div", { class: "barcode-icon" }, [
        el("i", { "data-lucide": "scan-barcode" }),
      ]),
      input,
      el(
        "button",
        {
          class: "btn btn-soft",
          title: "Limpar campo",
          onclick: () => {
            input.value = "";
            input.focus();
            statusEl.textContent = "Aguardando leitura…";
          },
        },
        [el("i", { "data-lucide": "x" })],
      ),
    ]),
    statusEl,
    el("div", { class: "barcode-history" }, [
      el("div", { class: "barcode-history-header" }, [
        el("div", { class: "barcode-history-title muted", text: "Últimas leituras" }),
        clearHistoryBtn,
      ]),
      historyList,
    ]),
  ]);

  renderIcons(wrapper);

  function refreshHistory() {
    const history = BarcodeHistory.list()
      .sort((a, b) => b.timestamp.localeCompare(a.timestamp))
      .slice(0, 6);
    historyList.innerHTML = "";
    if (history.length === 0) {
      historyList.appendChild(
        el("li", { class: "muted", style: "padding:6px 0;font-size:0.82em" }, [
          "Nenhuma leitura ainda.",
        ]),
      );
      return;
    }
    for (const h of history) {
      const li = el(
        "li",
        { class: "barcode-history-item" + (h.found ? "" : " not-found") },
        [
          el("span", { class: "barcode-code", text: h.code }),
          el("span", {
            class: "barcode-product",
            text: h.found ? h.itemName : "Não encontrado",
          }),
          el("span", {
            class: "barcode-time muted",
            text: h.timestamp.slice(11, 19),
          }),
        ],
      );
      historyList.appendChild(li);
    }
  }

  refreshHistory();

  function handleCode(code) {
    code = code.trim();
    if (code.length < SCANNER_MIN_LEN) return;

    statusEl.textContent = `Lido: ${code}`;
    statusEl.className = "barcode-status scanning";

    BarcodeHistory.save({
      id: Date.now().toString(),
      code,
      timestamp: new Date().toISOString(),
      found: false,
      itemName: "",
    });

    onScan?.({ code, refresh: refreshHistory });

    input.value = "";
    setTimeout(() => {
      statusEl.className = "barcode-status muted";
      statusEl.textContent = "Aguardando leitura…";
    }, 2000);
  }

  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleCode(input.value);
    }
  });

  // ── Captura global em segundo plano (detecção por velocidade de digitação) ──
  if (background) {
    let globalBuffer = "";
    let globalTimer = null;
    document.addEventListener("keydown", (e) => {
      if (
        document.activeElement !== document.body &&
        document.activeElement !== wrapper
      )
        return;
      const now = Date.now();
      if (now - lastKeyTime > 300) globalBuffer = "";
      lastKeyTime = now;
      if (e.key === "Enter") {
        if (globalBuffer.length >= SCANNER_MIN_LEN) handleCode(globalBuffer);
        globalBuffer = "";
        return;
      }
      if (e.key.length === 1) globalBuffer += e.key;
      clearTimeout(globalTimer);
      globalTimer = setTimeout(() => {
        globalBuffer = "";
      }, 150);
    });
  }

  if (autoFocus) setTimeout(() => input.focus(), 80);

  return { node: wrapper, input, refreshHistory };
}
