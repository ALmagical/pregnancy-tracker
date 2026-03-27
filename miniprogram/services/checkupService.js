import { ok, fail } from "../utils/errors";
import { Storage } from "../utils/storage";
import { STORAGE_KEYS } from "../utils/constants";
import { nowIso } from "../utils/date";

function getAll() {
  return Storage.get(STORAGE_KEYS.checkups, []);
}

function setAll(list) {
  Storage.set(STORAGE_KEYS.checkups, list);
}

function genId() {
  return `c_${Date.now()}`;
}

export async function listCheckupsApi({ page = 1, pageSize = 10 } = {}) {
  const all = getAll()
    .slice()
    .sort((a, b) => String(b.checkupDate).localeCompare(String(a.checkupDate)));
  const total = all.length;
  const start = (page - 1) * pageSize;
  const list = all.slice(start, start + pageSize);
  return ok({
    list,
    pagination: {
      page,
      pageSize,
      total,
      totalPages: Math.max(1, Math.ceil(total / pageSize))
    }
  });
}

export async function getCheckupDetailApi(id) {
  const all = getAll();
  const found = all.find((x) => x.id === id);
  if (!found) return fail("产检记录不存在", "E_NOT_FOUND", {});
  return ok(found);
}

export async function createCheckupApi(payload) {
  if (!payload?.checkupDate || !payload?.checkupType) {
    return fail("请填写产检日期与类型", "E_PARAM_INVALID", {});
  }
  const all = getAll();
  const item = {
    id: genId(),
    checkupDate: payload.checkupDate,
    checkupType: payload.checkupType,
    checkupTypeId: payload.checkupTypeId || "",
    hospital: payload.hospital || "",
    status: payload.status || "pending",
    hasReport: false,
    reportCount: 0,
    summary: payload.summary || "",
    note: payload.note || "",
    images: [],
    createdAt: nowIso(),
    updatedAt: nowIso()
  };
  all.push(item);
  setAll(all);
  return ok({ id: item.id, reminderId: `r_${Date.now()}` });
}

export async function updateCheckupApi(id, patch) {
  const all = getAll();
  const idx = all.findIndex((x) => x.id === id);
  if (idx < 0) return fail("产检记录不存在", "E_NOT_FOUND", {});
  all[idx] = { ...all[idx], ...patch, updatedAt: nowIso() };
  setAll(all);
  return ok({ id });
}

export async function deleteCheckupApi(id) {
  const all = getAll();
  const next = all.filter((x) => x.id !== id);
  if (next.length === all.length) return fail("产检记录不存在", "E_NOT_FOUND", {});
  setAll(next);
  return ok({});
}

export async function addCheckupReportsApi(id, { images = [], summary = "", note = "" } = {}) {
  const all = getAll();
  const idx = all.findIndex((x) => x.id === id);
  if (idx < 0) return fail("产检记录不存在", "E_NOT_FOUND", {});
  if (!Array.isArray(images) || images.length === 0) return fail("请选择要上传的图片", "E_PARAM_INVALID", {});
  if (images.length > 9) return fail("最多上传9张图片", "E_PARAM_INVALID", {});

  const current = all[idx];
  const mapped = images.map((path) => ({
    id: `img_${Date.now()}_${Math.random().toString(16).slice(2, 8)}`,
    url: path,
    thumbnail: path
  }));
  const nextImages = [...(current.images || []), ...mapped].slice(0, 9);
  all[idx] = {
    ...current,
    images: nextImages,
    hasReport: nextImages.length > 0,
    reportCount: nextImages.length,
    summary: summary || current.summary || "",
    note: note || current.note || "",
    updatedAt: nowIso()
  };
  setAll(all);
  return ok({ images: mapped, summary: all[idx].summary });
}

