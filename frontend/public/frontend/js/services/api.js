/**
 * api.js — Camada única de comunicação com o backend Go (wms-backend).
 *
 * Todas as chamadas HTTP do frontend passam por aqui. Nenhuma outra parte
 * do app monta URLs ou faz fetch diretamente — isso mantém o contrato com
 * a API em um único lugar, fácil de auditar.
 *
 * Regra de segurança: o frontend NUNCA decide permissões. Toda checagem de
 * "quem pode ver/editar o quê" acontece no backend; aqui apenas mostramos
 * ou escondemos botões por UX, sabendo que o backend recusaria de qualquer
 * forma uma chamada não autorizada.
 */
import { session } from "./store.js";

// Em desenvolvimento local: deixe como está (usa localhost:8000).
// Em produção: defina window.ANT_API_BASE_URL antes de carregar este script
// (ex.: <script>window.ANT_API_BASE_URL = "https://api.minhaescola.com/api/v1";</script>)
const BASE_URL = window.ANT_API_BASE_URL || "http://localhost:8000/api/v1";

/**
 * request — wrapper único sobre fetch().
 * Fluxo: injeta o JWT (se houver sessão), serializa o corpo em JSON,
 * e converte respostas não-2xx em Error com a mensagem vinda do backend.
 * Efeito colateral: nenhum além da chamada de rede em si.
 */
async function request(path, { method = "GET", body, params } = {}) {
  let url = BASE_URL + path;
  if (params) {
    const query = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== undefined && v !== null && v !== ""),
    ).toString();
    if (query) url += "?" + query;
  }

  const headers = { "Content-Type": "application/json" };
  if (session.token) headers.Authorization = "Bearer " + session.token;

  let response;
  try {
    response = await fetch(url, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new Error("Não foi possível conectar ao servidor. Verifique sua conexão.");
  }

  const isJson = response.headers.get("content-type")?.includes("application/json");
  const data = isJson ? await response.json().catch(() => null) : null;

  if (!response.ok) {
    if (response.status === 401) session.signOut();
    throw new Error(data?.error || `Erro ${response.status} ao comunicar com o servidor.`);
  }

  return data;
}

export const API = {
  // ── Autenticação ─────────────────────────────────────────────
  async login(email, password) {
    const data = await request("/auth/login", { method: "POST", body: { email, password } });
    return { user: normalizeUser(data.user), token: data.token };
  },

  /** Cria uma solicitação de conta PENDENTE. Não autentica automaticamente. */
  async register({ name, email, password, role }) {
    return request("/auth/register", { method: "POST", body: { name, email, password, role } });
  },

  /** Somente gestão: lista contas aguardando aprovação. */
  async pendingUsers() {
    const list = await request("/auth/pending");
    return list || [];
  },

  /** Somente gestão: aprova ("approve") ou rejeita ("reject") uma solicitação. */
  async approveUser(userId, action) {
    return request("/auth/approve", { method: "POST", body: { user_id: userId, action } });
  },

  // ── Usuário atual / lista de usuários ───────────────────────
  async me() {
    const user = await request("/users/me");
    return normalizeUser(user);
  },

  /** Somente gestão: lista todas as contas ativas. */
  async users() {
    const list = await request("/users");
    return (list || []).map(normalizeUser);
  },

  // ── Depósitos (estoques) ────────────────────────────────────
  async deposits() {
    const list = await request("/deposits");
    return list || [];
  },
  async createDeposit({ name, description }) {
    return request("/deposits", { method: "POST", body: { name, description } });
  },
  async updateDeposit(id, { name, description }) {
    return request(`/deposits/${id}`, { method: "PATCH", body: { name, description } });
  },
  async deleteDeposit(id) {
    return request(`/deposits/${id}`, { method: "DELETE" });
  },

  // ── Inventário (itens de estoque) ───────────────────────────
  async inventory(depositId) {
    const list = await request("/inventory", { params: { deposit_id: depositId } });
    return list || [];
  },
  async createInventoryItem({ depositId, name, sku, minQuantity }) {
    return request("/inventory", {
      method: "POST",
      body: { deposit_id: depositId, name, sku, min_quantity: minQuantity },
    });
  },
  async updateInventoryItem(id, { name, sku, minQuantity }) {
    return request(`/inventory/${id}`, {
      method: "PATCH",
      body: { name, sku, min_quantity: minQuantity },
    });
  },
  async deleteInventoryItem(id) {
    return request(`/inventory/${id}`, { method: "DELETE" });
  },

  /** Registra uma entrada ("in") ou saída ("out") de estoque. */
  async moveStock({ inventoryItemId, type, quantity, note }) {
    return request("/inventory/move", {
      method: "POST",
      body: { inventory_item_id: inventoryItemId, type, quantity, note },
    });
  },
  async movements({ depositId } = {}) {
    const list = await request("/inventory/movements", { params: { deposit_id: depositId } });
    return list || [];
  },

  // ── Turmas ───────────────────────────────────────────────────
  async classes() {
    const list = await request("/classes");
    return list || [];
  },
  async createClass({ name, description, teacherIds, depositIds }) {
    return request("/classes", {
      method: "POST",
      body: { name, description, teacher_ids: teacherIds, deposit_ids: depositIds },
    });
  },
  async updateClass(id, { name, description, teacherIds, depositIds }) {
    return request(`/classes/${id}`, {
      method: "PATCH",
      body: { name, description, teacher_ids: teacherIds, deposit_ids: depositIds },
    });
  },
  async deleteClass(id) {
    return request(`/classes/${id}`, { method: "DELETE" });
  },
};

/**
 * normalizeUser — adapta o usuário vindo da API para o formato usado
 * internamente pelo frontend (mantém `role` como fonte da verdade;
 * nenhuma outra tela deve inventar seu próprio rótulo de perfil).
 */
export function normalizeUser(u) {
  if (!u) return u;
  return {
    id: u.id,
    name: u.name,
    email: u.email,
    role: u.role, // "professor" | "gestao"
    active: u.active,
    classes: u.classes || [],
    deposits: u.deposits || [],
    createdAt: u.created_at,
  };
}

export const ROLE_LABEL = { professor: "Professor", gestao: "Gestão" };
