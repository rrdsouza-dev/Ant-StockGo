/** Tiny pub/sub store + localStorage-backed session. */
const KEY = "antstock:session";

const listeners = new Set();
let state = load();

function load() {
  try { return JSON.parse(localStorage.getItem(KEY) || "null") || { user: null, token: null, depositId: null, classId: null }; }
  catch { return { user: null, token: null, depositId: null, classId: null }; }
}
function save() { localStorage.setItem(KEY, JSON.stringify(state)); }

export const session = {
  get user() { return state.user; },
  get token() { return state.token; },
  get depositId() { return state.depositId; },
  /** Turma "ativa" escolhida pelo professor na sessão atual (ver choose-class.js). */
  get classId() { return state.classId; },
  isAuthenticated() { return !!state.user && !!state.token; },
  signIn(user, token = state.token) { state = { user, token, depositId: state.depositId, classId: state.classId }; save(); emit(); },
  signOut() { state = { user: null, token: null, depositId: null, classId: null }; save(); emit(); },
  setDepositId(id) { state = { ...state, depositId: id }; save(); emit(); },
  /** Define a turma ativa (ou null para "trocar de turma" / limpar a escolha). */
  setClassId(id) { state = { ...state, classId: id, depositId: null }; save(); emit(); },
  subscribe(fn) { listeners.add(fn); return () => listeners.delete(fn); },
};

function emit() { for (const fn of listeners) fn(state); }
