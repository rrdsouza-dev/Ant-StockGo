/**
 * aiIcon.js — ícone de estrela usado para sinalizar funcionalidades de
 * IA no ANT-Stock (Otis, Importação Inteligente). Desenhado à mão (sem
 * depender de um ícone Lucide específico), com traço arredondado para
 * casar com o strokeWidth 1.8 usado no resto do sistema.
 *
 * Extraído de components/otis.js para um módulo compartilhado — antes
 * só existia ali; agora ambos os pontos de IA do sistema usam a mesma
 * fonte, em vez de duas definições do mesmo SVG divergindo com o tempo.
 */
export function starIcon(size = 22, className = "otis-star") {
  const ns = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(ns, "svg");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("width", String(size));
  svg.setAttribute("height", String(size));
  svg.setAttribute("fill", "none");
  if (className) svg.classList.add(className);
  const path = document.createElementNS(ns, "path");
  path.setAttribute(
    "d",
    "M12 2.5c.35 0 .66.23.76.57l1.53 5.12a3.6 3.6 0 0 0 2.42 2.42l5.12 1.53a.8.8 0 0 1 0 1.53l-5.12 1.53a3.6 3.6 0 0 0-2.42 2.42l-1.53 5.12a.8.8 0 0 1-1.53 0l-1.53-5.12a3.6 3.6 0 0 0-2.42-2.42L2.16 13.7a.8.8 0 0 1 0-1.53l5.12-1.53a3.6 3.6 0 0 0 2.42-2.42l1.53-5.12A.79.79 0 0 1 12 2.5Z"
  );
  path.setAttribute("fill", "currentColor");
  svg.appendChild(path);
  return svg;
}
