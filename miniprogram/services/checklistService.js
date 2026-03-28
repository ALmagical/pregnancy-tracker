import { useRemoteApi } from "../config";
import { ok, fail } from "../utils/errors";
import { apiGet, apiPost, apiPut } from "../utils/apiClient";
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

/** 将设计文档 6.8.1 的分组结构转为页面使用的平铺 items */
function checklistApiToLocal(apiData) {
  if (!apiData || typeof apiData !== "object") {
    return { items: [], version: 1, updatedAt: nowIso() };
  }
  const items = [];
  const cats = apiData.categories || [];
  for (let i = 0; i < cats.length; i += 1) {
    const cat = cats[i];
    const catId = cat.id || `cat_${i}`;
    const list = cat.items || [];
    for (let j = 0; j < list.length; j += 1) {
      const it = list[j];
      items.push({
        id: it.id,
        title: it.title,
        checked: !!it.checked,
        note: it.note != null ? String(it.note) : "",
        categoryId: catId,
        source: it.source || "template"
      });
    }
  }
  return {
    items,
    version: apiData.version ?? 1,
    updatedAt: nowIso(),
    progress: apiData.progress
  };
}

export async function getChecklistApi() {
  if (useRemoteApi() && isOnline()) {
    const r = await apiGet("/api/v1/checklist");
    if (r.code !== 0) return r;
    const localShape = checklistApiToLocal(r.data);
    setState(localShape);
    return ok(localShape);
  }
  if (useRemoteApi() && !isOnline()) {
    return ok(getState());
  }
  return ok(getState());
}

export async function toggleChecklistItemApi(id, checked) {
  const s = getState();
  const idx = s.items.findIndex((x) => x.id === id);
  if (idx < 0) return fail("清单项不存在", "E_NOT_FOUND", {});

  if (useRemoteApi() && isOnline()) {
    const item = s.items[idx];
    const r = await apiPut(`/api/v1/checklist/items/${encodeURIComponent(id)}`, {
      checked: !!checked,
      note: item.note || ""
    });
    if (r.code !== 0) return r;
    const nextItems = s.items.slice();
    nextItems[idx] = { ...nextItems[idx], checked: !!checked };
    const next = { ...s, items: nextItems, updatedAt: nowIso() };
    setState(next);
    return ok(next);
  }

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

  const cat = categoryId || "custom";

  if (useRemoteApi() && isOnline()) {
    const r = await apiPost("/api/v1/checklist/items", { categoryId: cat, title: t, note: n });
    if (r.code !== 0) return r;
    await getChecklistApi();
    return ok(getState());
  }

  const s = getState();
  const item = {
    id: genId(),
    title: t,
    note: n,
    categoryId: cat,
    source: "custom",
    checked: false
  };
  const next = { ...s, items: [item, ...s.items], updatedAt: nowIso() };
  setState(next);
  if (!isOnline()) enqueueOfflineAction({ type: "checklist.add", payload: item });
  return ok(next);
}

export async function resetChecklistApi() {
  if (useRemoteApi() && isOnline()) {
    const r = await apiPost("/api/v1/checklist/reset", { keepCustomItems: false });
    if (r.code !== 0) return r;
    await getChecklistApi();
    return ok(getState());
  }

  const next = {
    items: DEFAULT_TEMPLATE.map((x) => ({ ...x, checked: false, note: "" })),
    updatedAt: nowIso(),
    version: 1
  };
  setState(next);
  if (!isOnline()) enqueueOfflineAction({ type: "checklist.reset", payload: {} });
  return ok(next);
}
