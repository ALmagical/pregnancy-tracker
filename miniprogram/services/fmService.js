import { ok, fail } from "../utils/errors";
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

export async function getActiveFmSessionApi() {
  return ok({ active: getActive() });
}

export async function startFmSessionApi() {
  const active = getActive();
  if (active) return fail("已有进行中的胎动计数", "E_FM_SESSION_RUNNING", {});
  const s = { id: genId(), startedAt: nowIso(), count: 0 };
  setActive(s);
  return ok({ active: s });
}

export async function incFmCountApi(delta = 1) {
  const active = getActive();
  if (!active) return fail("请先开始计数", "E_NOT_FOUND", {});
  const next = { ...active, count: Math.max(0, Math.min(999, active.count + delta)) };
  setActive(next);
  return ok({ active: next });
}

export async function finishFmSessionApi() {
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
  const all = getSessions()
    .slice()
    .sort((a, b) => String(b.startedAt).localeCompare(String(a.startedAt)));
  return ok({ list: all });
}

