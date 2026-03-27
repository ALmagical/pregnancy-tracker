import { ok, fail } from "../utils/errors";
import { Storage, enqueueOfflineAction } from "../utils/storage";
import { STORAGE_KEYS } from "../utils/constants";
import { nowIso } from "../utils/date";
import { isOnline } from "../utils/net";

function getAll() {
  return Storage.get(STORAGE_KEYS.contractions, []);
}

function setAll(list) {
  Storage.set(STORAGE_KEYS.contractions, list);
}

function genId() {
  return `ct_${Date.now()}`;
}

export async function listContractionsApi() {
  const all = getAll()
    .slice()
    .sort((a, b) => String(b.startedAt).localeCompare(String(a.startedAt)));
  return ok({ list: all });
}

export async function createContractionApi({ startedAt, endedAt } = {}) {
  if (!startedAt || !endedAt) return fail("请填写开始与结束时间", "E_PARAM_INVALID", {});
  const s = Date.parse(startedAt);
  const e = Date.parse(endedAt);
  if (!Number.isFinite(s) || !Number.isFinite(e) || e < s) return fail("结束时间不能早于开始时间", "E_CONTRACTION_TIME_INVALID", {});

  const durationSec = Math.floor((e - s) / 1000);
  const item = {
    id: genId(),
    startedAt,
    endedAt,
    durationSec,
    createdAt: nowIso(),
    syncStatus: isOnline() ? "synced" : "pending"
  };
  const all = getAll();
  all.push(item);
  setAll(all);
  if (!isOnline()) enqueueOfflineAction({ type: "contraction.create", payload: item });
  return ok({ id: item.id });
}

