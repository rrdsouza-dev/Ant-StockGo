export const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

export const isEmail = (v) => emailRegex.test(String(v || "").trim());

// Espelha exatamente a regra validada no backend (ver internal/validation).
// Mantido apenas para feedback de UX — a validação que vale é sempre a do servidor.
export function passwordScore(pw) {
  let s = 0;
  if (!pw) return 0;
  if (pw.length >= 8) s++;
  if (/[A-Z]/.test(pw) && /[a-z]/.test(pw)) s++;
  if (/\d/.test(pw)) s++;
  if (/[^A-Za-z0-9]/.test(pw)) s++;
  return s; // 0..4
}

export function validatePassword(pw) {
  if (!pw) return "Informe a senha.";
  if (pw.length < 8) return "A senha deve ter ao menos 8 caracteres.";
  if (!/[A-Z]/.test(pw)) return "Inclua ao menos uma letra maiúscula.";
  if (!/\d/.test(pw)) return "Inclua ao menos um número.";
  if (!/[^A-Za-z0-9]/.test(pw)) return "Inclua ao menos um símbolo (ex.: ! @ # $).";
  return null;
}

export function validateEmail(v) {
  if (!v) return "Informe seu e-mail.";
  if (!isEmail(v)) return "E-mail inválido.";
  return null;
}

export function required(v, label = "Campo") {
  if (!String(v ?? "").trim()) return `${label} é obrigatório.`;
  return null;
}

// ── Data de validade (DD/MM/AAAA) ──────────────────────────────
// Mensagens divertidas exigidas pela especificação para datas inválidas.
// Sorteada a cada erro — ver InventoryModal.
export const FUNNY_DATE_ERRORS = [
  "Tá chapando fiot? kkkkk",
  "Presta atenção mano kkkk",
  "Xiiiii... o negócio tá feio hein kkkk",
];

export function randomFunnyDateError() {
  return FUNNY_DATE_ERRORS[Math.floor(Math.random() * FUNNY_DATE_ERRORS.length)];
}

/** Aplica a máscara 00/00/0000 enquanto o usuário digita. */
export function applyDateMask(raw) {
  const digits = String(raw || "").replace(/\D/g, "").slice(0, 8);
  let out = digits;
  if (digits.length > 4) out = `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
  else if (digits.length > 2) out = `${digits.slice(0, 2)}/${digits.slice(2)}`;
  return out;
}

/**
 * Valida uma data no formato DD/MM/AAAA de forma manual (sem confiar em
 * `new Date(...)`, que normaliza datas inexistentes como 31/02 em vez de
 * rejeitá-las). Espelha internal/validation.ParseBRDate do backend — a
 * validação que realmente vale é sempre a do servidor; esta é só para dar
 * feedback imediato (com a mensagem divertida pedida na especificação).
 */
export function isValidBRDate(value) {
  const m = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(String(value || "").trim());
  if (!m) return false;
  const day = Number(m[1]);
  const month = Number(m[2]);
  const year = Number(m[3]);
  if (month < 1 || month > 12) return false;
  if (day < 1 || day > 31) return false;
  const daysInMonth = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  const isLeap = (year % 4 === 0 && year % 100 !== 0) || year % 400 === 0;
  const max = month === 2 && isLeap ? 29 : daysInMonth[month - 1];
  return day <= max;
}
