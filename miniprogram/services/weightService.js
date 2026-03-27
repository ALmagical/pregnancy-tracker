import { ok, fail } from "../utils/errors";
import { Storage, enqueueOfflineAction } from "../utils/storage";
import { STORAGE_KEYS } from "../utils/constants";
import { nowIso } from "../utils/date";
import { validateWeightKg } from "../utils/validation";
import { isOnline } from "../utils/net";

function getAll() {
  return Storage.get(STORAGE_KEYS.weights, []);
}

function setAll(list) {
  Storage.set(STORAGE_KEYS.weights, list);
}

function genId() {
  return `w_${Date.now()}`;
}

export async function listWeightsApi({ page = 1, pageSize = 30 } = {}) {
  const all = getAll()
    .slice()
    .sort((a, b) => String(b.recordedAt).localeCompare(String(a.recordedAt)));
  const total = all.length;
  const start = (page - 1) * pageSize;
  const list = all.slice(start, start + pageSize);

  const currentWeight = list.length ? list[0].weight : null;
  const prePregnancyWeight = null;
  const totalGain = prePregnancyWeight != null && currentWeight != null ? currentWeight - prePregnancyWeight : null;

  return ok({
    list,
    statistics: {
      currentWeight,
      prePregnancyWeight,
      totalGain,
      averageWeeklyGain: null,
      recommendedRange: { min: null, max: null }
    }
  });
}

export async function createWeightApi({ weight, week, day, recordedAt } = {}) {
  const v = validateWeightKg(weight);
  if (!v.ok && v.errorCode !== "E_WEIGHT_UNREASONABLE") return fail(v.message, v.errorCode, {});
  const rounded = v?.data?.rounded ?? Math.round(Number(weight) * 10) / 10;

  const item = {
    id: genId(),
    weight: rounded,
    recordedAt: recordedAt || nowIso(),
    week: typeof week === "number" ? week : null,
    day: typeof day === "number" ? day : null,
    syncStatus: isOnline() ? "synced" : "pending"
  };

  const all = getAll();
  all.push(item);
  setAll(all);

  if (!isOnline()) {
    enqueueOfflineAction({ type: "weight.create", payload: item });
  }

  return ok({ id: item.id });
}

export async function deleteWeightApi(id) {
  const all = getAll();
  const next = all.filter((x) => x.id !== id);
  if (next.length === all.length) return fail("记录不存在", "E_NOT_FOUND", {});
  setAll(next);
  if (!isOnline()) enqueueOfflineAction({ type: "weight.delete", payload: { id } });
  return ok({});
}

