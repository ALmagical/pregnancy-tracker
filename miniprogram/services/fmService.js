import { FM_SESSIONS_LIST_PATH, useRemoteApi } from "../config";
import { ok, fail } from "../utils/errors";
import { apiGet, apiPost } from "../utils/apiClient";
import { Storage, enqueueOfflineAction } from "../utils/storage";
import { STORAGE_KEYS } from "../utils/constants";
import { nowIso } from "../utils/date";
import { isOnline } from "../utils/net";

function getSessions() {
  return Storage.get(STORAGE_KEYS.fmSessions, []);
}

function setSessions(list) {
  Storage.set(STORAGE_KEYS.fmSessions, list);
}

function getActive() {
  return Storage.get(STORAGE_KEYS.fmActiveSession, null);
}

function setActive(active) {
  Storage.set(STORAGE_KEYS.fmActiveSession, active);
}

function genId() {
  return `fm_${Date.now()}`;
}

function mapFinishResponse(data, fallbackActive) {
  if (!data || typeof data !== "object") {
    return {
      id: fallbackActive?.id || genId(),
      startedAt: fallbackActive?.startedAt || nowIso(),
      endedAt: nowIso(),
      count: fallbackActive?.count ?? 0,
      durationSec: 0,
      syncStatus: "synced"
    };
  }
  return {
    id: data.id || fallbackActive?.id,
    startedAt: fallbackActive?.startedAt || nowIso(),
    endedAt: nowIso(),
    count: typeof data.count === "number" ? data.count : fallbackActive?.count ?? 0,
    durationSec: typeof data.durationSec === "number" ? data.durationSec : 0,
    syncStatus: "synced"
  };
}

export async function getActiveFmSessionApi() {
  return ok({ active: getActive() });
}

export async function startFmSessionApi() {
  if (useRemoteApi()) {
    if (!isOnline()) return fail("使用云端服务时请联网后再开始胎动计数", "E_NETWORK", {});
    const active = getActive();
    if (active) return fail("已有进行中的胎动计数", "E_FM_SESSION_RUNNING", {});
    const r = await apiPost("/api/v1/fetal-movements/sessions", { startedAt: nowIso(), note: "" });
    if (r.code !== 0) return r;
    const sid = r.data?.id;
    if (!sid) return fail("服务端未返回会话 id", "E_PARAM_INVALID", {});
    const s = { id: sid, startedAt: r.data?.startedAt || nowIso(), count: 0 };
    setActive(s);
    return ok({ active: s });
  }

  const active = getActive();
  if (active) return fail("已有进行中的胎动计数", "E_FM_SESSION_RUNNING", {});
  const s = { id: genId(), startedAt: nowIso(), count: 0 };
  setActive(s);
  return ok({ active: s });
}

export async function incFmCountApi(delta = 1) {
  if (useRemoteApi()) {
    if (!isOnline()) return fail("请联网后再操作", "E_NETWORK", {});
    const active = getActive();
    if (!active) return fail("请先开始计数", "E_NOT_FOUND", {});
    const type = delta >= 0 ? "add" : "undo";
    const times = Math.min(20, Math.abs(Math.trunc(delta)) || 1);
    let last = null;
    for (let i = 0; i < times; i += 1) {
      last = await apiPost(`/api/v1/fetal-movements/sessions/${encodeURIComponent(active.id)}/events`, { type });
      if (last.code !== 0) return last;
    }
    const count = typeof last.data?.count === "number" ? last.data.count : active.count + (delta >= 0 ? times : -times);
    const next = { ...active, count: Math.max(0, Math.min(999, count)) };
    setActive(next);
    return ok({ active: next });
  }

  const active = getActive();
  if (!active) return fail("请先开始计数", "E_NOT_FOUND", {});
  const next = { ...active, count: Math.max(0, Math.min(999, active.count + delta)) };
  setActive(next);
  return ok({ active: next });
}

export async function finishFmSessionApi() {
  if (useRemoteApi()) {
    if (!isOnline()) return fail("请联网后再保存胎动记录", "E_NETWORK", {});
    const active = getActive();
    if (!active) return fail("没有进行中的计数", "E_NOT_FOUND", {});
    const endedAt = nowIso();
    const r = await apiPost(`/api/v1/fetal-movements/sessions/${encodeURIComponent(active.id)}/finish`, {
      endedAt,
      resultTag: "normal"
    });
    if (r.code !== 0) return r;
    const session = mapFinishResponse(r.data, active);
    session.startedAt = active.startedAt;
    session.endedAt = endedAt;
    session.count = typeof r.data?.count === "number" ? r.data.count : active.count;
    const all = getSessions();
    all.push(session);
    setSessions(all);
    setActive(null);
    return ok({ session });
  }

  const active = getActive();
  if (!active) return fail("没有进行中的计数", "E_NOT_FOUND", {});
  const endedAt = nowIso();
  const session = {
    ...active,
    endedAt,
    durationSec: Math.max(0, Math.floor((Date.parse(endedAt) - Date.parse(active.startedAt)) / 1000)),
    syncStatus: isOnline() ? "synced" : "pending"
  };
  const all = getSessions();
  all.push(session);
  setSessions(all);
  setActive(null);
  if (!isOnline()) enqueueOfflineAction({ type: "fm.finish", payload: session });
  return ok({ session });
}

export async function listFmSessionsApi() {
  if (useRemoteApi()) {
    if (!isOnline()) {
      const all = getSessions()
        .slice()
        .sort((a, b) => String(b.startedAt).localeCompare(String(a.startedAt)));
      return ok({ list: all });
    }
    const r = await apiGet(FM_SESSIONS_LIST_PATH, { limit: 50 });
    if (r.code !== 0) return ok({ list: [] });
    return ok({ list: r.data?.list || [] });
  }

  const all = getSessions()
    .slice()
    .sort((a, b) => String(b.startedAt).localeCompare(String(a.startedAt)));
  return ok({ list: all });
}
