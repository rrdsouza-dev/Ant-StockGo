export function exportTxt(rows, filename = "export.txt") {
  if (!Array.isArray(rows) || rows.length === 0) rows = [{ info: "Nenhum dado para exportar." }];
  const headers = Object.keys(rows[0]);
  const lines = [headers.join("\t")];
  for (const r of rows) {
    lines.push(headers.map((h) => String(r[h] ?? "").replace(/\t|\n|\r/g, " ")).join("\t"));
  }
  const blob = new Blob([lines.join("\n")], { type: "text/plain;charset=utf-8" });
  triggerDownload(blob, filename);
}

export function triggerDownload(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url; a.download = filename;
  document.body.appendChild(a); a.click();
  setTimeout(() => { URL.revokeObjectURL(url); a.remove(); }, 0);
}