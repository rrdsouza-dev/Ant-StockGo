/** Tiny pub/sub store + localStorage-backed session. */
const KEY = "antstock:session";

const listeners = new Set();
let state = load();

function load() {
  try { return JSON.parse(localStorage.getItem(KEY) || "null") || { user: null, token: null, depositId: null }; }
  catch { return { user: null, token: null, depositId: null }; }
}
function save() { localStorage.setItem(KEY, JSON.stringify(state)); }

export const session = {
  get user() { return state.user; },
  get token() { return state.token; },
  get depositId() { return state.depositId; },
  isAuthenticated() { return !!state.user && !!state.token; },
  signIn(user, token = state.token) { state = { user, token, depositId: state.depositId }; save(); emit(); },
  signOut() { state = { user: null, token: null, depositId: null }; save(); emit(); },
  setDepositId(id) { state = { ...state, depositId: id }; save(); emit(); },
  subscribe(fn) { listeners.add(fn); return () => listeners.delete(fn); },
};

function emit() { for (const fn of listeners) fn(state); }
