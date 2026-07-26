/**
 * period.js — cálculo de intervalos de data para os filtros de período
 * usados nas telas de Entradas/Saídas e Relatórios. Cada preset resolve
 * para datas no formato AAAA-MM-DD, o mesmo aceito pela API em
 * GET /inventory/movements?from=&to=.
 *
 * O histórico de movimentações nunca é apagado no banco — estes filtros
 * são só uma restrição de consulta/visualização.
 */

export const PERIOD_PRESETS = [
  { value: "today", label: "Hoje" },
  { value: "7d", label: "Últimos 7 dias" },
  { value: "30d", label: "Últimos 30 dias" },
  { value: "custom", label: "Período personalizado" },
];

function toISODate(date) {
  return date.toISOString().slice(0, 10);
}

/**
 * Resolve um preset em { from, to } (strings AAAA-MM-DD). Para "custom",
 * retorna null — o chamador deve usar os campos de data escolhidos
 * manualmente pelo usuário nesse caso.
 */
export function resolvePeriod(preset) {
  const today = new Date();
  const to = toISODate(today);

  if (preset === "today") return { from: to, to };

  if (preset === "7d") {
    const from = new Date(today);
    from.setDate(from.getDate() - 6);
    return { from: toISODate(from), to };
  }

  if (preset === "30d") {
    const from = new Date(today);
    from.setDate(from.getDate() - 29);
    return { from: toISODate(from), to };
  }

  return null;
}
