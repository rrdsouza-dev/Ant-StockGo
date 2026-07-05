export function exportExcel(rows, filename = "export.xlsx", sheetName = "Dados") {
  if (!window.XLSX) { console.warn("SheetJS not loaded"); return; }
  const ws = XLSX.utils.json_to_sheet(rows.length ? rows : [{ info: "Sem dados" }]);
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, sheetName);
  XLSX.writeFile(wb, filename);
}