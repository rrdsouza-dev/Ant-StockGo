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
