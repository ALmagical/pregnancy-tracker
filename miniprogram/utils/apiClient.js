import { API_BASE_URL } from "../config";
import { request } from "./request";
import { fail } from "./errors";
import { getAuthToken } from "./storage";
import { ensureWechatSession } from "../services/authService";
import { useRemoteApi } from "../config";

export function buildApiUrl(path) {
  const base = String(API_BASE_URL || "").replace(/\/+$/, "");
  const p = path.startsWith("/") ? path : `/${path}`;
  return `${base}${p}`;
}

function mergeQuery(path, query) {
  if (!query || typeof query !== "object") return path;
  const sp = new URLSearchParams();
  Object.keys(query).forEach((k) => {
    const v = query[k];
    if (v === undefined || v === null || v === "") return;
    sp.set(k, String(v));
  });
  const s = sp.toString();
  if (!s) return path;
  return `${path}${path.includes("?") ? "&" : "?"}${s}`;
}

export async function apiCall(path, { method = "GET", data, header = {}, query } = {}) {
  const url = buildApiUrl(mergeQuery(path, query));
  const h = { "Content-Type": "application/json", ...header };
  let token = getAuthToken();
  if (!token && useRemoteApi()) {
    // 远程模式下若尚未拿到 token，先尝试静默登录一次
    const ok = await ensureWechatSession();
    if (ok) token = getAuthToken();
  }
  if (token) h.Authorization = `Bearer ${token}`;
  const raw = await request({ url, method, header: h, data: method === "GET" ? undefined : data });
  if (raw && typeof raw.code === "number" && raw.code !== 0) {
    return fail(raw.message || "请求失败", raw.errorCode || "E_PARAM_INVALID", raw.data || {}, raw.code);
  }
  return raw;
}

export function apiGet(path, query) {
  return apiCall(path, { method: "GET", query });
}

export function apiPost(path, data, query) {
  return apiCall(path, { method: "POST", data, query });
}

export function apiPut(path, data, query) {
  return apiCall(path, { method: "PUT", data, query });
}

export function apiDelete(path, query) {
  return apiCall(path, { method: "DELETE", query });
}
