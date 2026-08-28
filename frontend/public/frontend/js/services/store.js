/** Tiny pub/sub store + localStorage-backed session. */
const KEY = "antstock:session";

const listeners = new Set();
let state = load();

function load() {
  try { return JSON.parse(localStorage.getItem(KEY) || "null") || { user: null, token: null, depositId: null, classId: null }; }
  catch { return { user: null, token: null, depositId: null, classId: null }; }
}
function save() { localStorage.setItem(KEY, JSON.stringify(state)); }

/**
 * decodeJWTPayload — lê o payload de um JWT (base64url) sem verificar
 * assinatura. Uso EXCLUSIVAMENTE de UX no frontend (decidir se vale a
 * pena nem tentar mostrar uma tela autenticada com um token já vencido);
 * nunca é a fonte de verdade de autenticação — isso é sempre o backend,
 * que valida assinatura e expiração de novo em toda requisição (ver
 * middleware.RequireAuth).
 */
function decodeJWTPayload(token) {
  try {
    const [, payload] = token.split(".");
    if (!payload) return null;
    const base64 = payload.replace(/-/g, "+").replace(/_/g, "/");
    const json = decodeURIComponent(
      atob(base64).split("").map((c) => "%" + c.charCodeAt(0).toString(16).padStart(2, "0")).join("")
    );
    return JSON.parse(json);
  } catch {
    return null;
  }
}

/**
 * isExpired — confirma se um JWT já passou do `exp` (com margem de 5s
 * para relógio). Token sem `exp` decodificável é tratado como expirado
 * (mais seguro do que assumir válido às cegas).
 */
function isExpired(token) {
  const claims = decodeJWTPayload(token);
  if (!claims?.exp) return true;
  return Date.now() >= claims.exp * 1000 - 5000;
}

export const session = {
  get user() { return state.user; },
  get token() { return state.token; },
  get depositId() { return state.depositId; },
  /** Turma "ativa" escolhida pelo professor na sessão atual (ver choose-class.js). */
  get classId() { return state.classId; },
  /**
   * isAuthenticated — além de exigir user+token salvos, confere se o
   * token já não expirou. Isso é o que evita o boot "fantasma": sem essa
   * checagem, reabrir o navegador depois da expiração fazia o guard de
   * rotas liberar o dashboard normalmente (user/token existiam no
   * localStorage), e só a primeira chamada à API revelava o problema via
   * 401 — sem nenhum redirecionamento automático de volta ao login (ver
   * api.js e app.js).
   */
  isAuthenticated() { return !!state.user && !!state.token && !isExpired(state.token); },
  signIn(user, token = state.token) { state = { user, token, depositId: state.depositId, classId: state.classId }; save(); emit(); },
  signOut() { state = { user: null, token: null, depositId: null, classId: null }; save(); emit(); },
  setDepositId(id) { state = { ...state, depositId: id }; save(); emit(); },
  /** Define a turma ativa (ou null para "trocar de turma" / limpar a escolha). */
  setClassId(id) { state = { ...state, classId: id, depositId: null }; save(); emit(); },
  subscribe(fn) { listeners.add(fn); return () => listeners.delete(fn); },
};

function emit() { for (const fn of listeners) fn(state); }
