import { ok, fail } from "../utils/errors";
import { Storage, enqueueOfflineAction } from "../utils/storage";
import { STORAGE_KEYS } from "../utils/constants";
import { nowIso } from "../utils/date";
import { isOnline } from "../utils/net";

const DEFAULT_TEMPLATE = [
  { id: "tpl_mom_1", title: "身份证/产检本", categoryId: "mom", source: "template" },
  { id: "tpl_mom_2", title: "住院缴费/银行卡", categoryId: "mom", source: "template" },
  { id: "tpl_baby_1", title: "纸尿裤", categoryId: "baby", source: "template" },
  { id: "tpl_baby_2", title: "包被/小毯子", categoryId: "baby", source: "template" }
];

function getState() {
  const s = Storage.get(STORAGE_KEYS.checklist, null);
  if (s) return s;
  const initial = {
    items: DEFAULT_TEMPLATE.map((x) => ({ ...x, checked: false, note: "" })),
    updatedAt: nowIso(),
    version: 1
  };
  Storage.set(STORAGE_KEYS.checklist, initial);
  return initial;
}

function setState(s) {
  Storage.set(STORAGE_KEYS.checklist, s);
}

function genId() {
  return `cl_${Date.now()}`;
}

export async function getChecklistApi() {
  return ok(getState());
}

export async function toggleChecklistItemApi(id, checked) {
  const s = getState();
  const idx = s.items.findIndex((x) => x.id === id);
  if (idx < 0) return fail("清单项不存在", "E_NOT_FOUND", {});
  const nextItems = s.items.slice();
  nextItems[idx] = { ...nextItems[idx], checked: !!checked };
  const next = { ...s, items: nextItems, updatedAt: nowIso() };
  setState(next);
  if (!isOnline()) enqueueOfflineAction({ type: "checklist.toggle", payload: { id, checked: !!checked } });
  return ok(next);
}

export async function addChecklistItemApi({ title, note, categoryId } = {}) {
  const t = String(title || "").trim();
  const n = String(note || "").trim();
  if (!t) return fail("请填写清单项名称", "E_PARAM_INVALID", {});
  if (t.length > 20 || n.length > 50) return fail("清单项长度超出限制", "E_CHECKLIST_ITEM_TOO_LONG", {});
  const s = getState();
  const item = {
    id: genId(),
    title: t,
    note: n,
    categoryId: categoryId || "custom",
    source: "custom",
    checked: false
  };
  const next = { ...s, items: [item, ...s.items], updatedAt: nowIso() };
  setState(next);
  if (!isOnline()) enqueueOfflineAction({ type: "checklist.add", payload: item });
  return ok(next);
}

export async function resetChecklistApi() {
  const next = {
    items: DEFAULT_TEMPLATE.map((x) => ({ ...x, checked: false, note: "" })),
    updatedAt: nowIso(),
    version: 1
  };
  setState(next);
  if (!isOnline()) enqueueOfflineAction({ type: "checklist.reset", payload: {} });
  return ok(next);
}

