import { useRemoteApi } from "../config";
import { ok, fail } from "../utils/errors";
import { apiDelete, apiGet, apiPost, apiPut, buildApiUrl } from "../utils/apiClient";
import { Storage, getAuthToken } from "../utils/storage";
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

function uploadReportsSequential(id, filePaths, summary, note) {
  const token = getAuthToken();
  const header = {};
  if (token) header.Authorization = `Bearer ${token}`;
  let last = null;
  return filePaths.reduce(
    (chain, filePath) =>
      chain.then(() => {
        return new Promise((resolve, reject) => {
          wx.uploadFile({
            url: buildApiUrl(`/api/v1/checkups/${encodeURIComponent(id)}/reports`),
            filePath,
            name: "images",
            header,
            formData: { summary: summary || "", note: note || "" },
            success(res) {
              try {
                last = JSON.parse(res.data || "{}");
              } catch (e) {
                last = { code: 10001, message: res.data || "上传解析失败" };
              }
              resolve();
            },
            fail: reject
          });
        });
      }),
    Promise.resolve()
  ).then(() => last);
}

export async function listCheckupsApi({ page = 1, pageSize = 10, status = "all" } = {}) {
  if (useRemoteApi()) {
    return apiGet("/api/v1/checkups", { page, pageSize, status });
  }
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
  if (useRemoteApi()) {
    return apiGet(`/api/v1/checkups/${encodeURIComponent(id)}`);
  }
  const all = getAll();
  const found = all.find((x) => x.id === id);
  if (!found) return fail("产检记录不存在", "E_NOT_FOUND", {});
  return ok(found);
}

export async function createCheckupApi(payload) {
  if (!payload?.checkupDate || !payload?.checkupType) {
    return fail("请填写产检日期与类型", "E_PARAM_INVALID", {});
  }
  if (useRemoteApi()) {
    return apiPost("/api/v1/checkups", {
      checkupDate: payload.checkupDate,
      checkupType: payload.checkupType,
      checkupTypeId: payload.checkupTypeId || "",
      hospital: payload.hospital || "",
      note: payload.note || ""
    });
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
  if (useRemoteApi()) {
    return apiPut(`/api/v1/checkups/${encodeURIComponent(id)}`, patch);
  }
  const all = getAll();
  const idx = all.findIndex((x) => x.id === id);
  if (idx < 0) return fail("产检记录不存在", "E_NOT_FOUND", {});
  all[idx] = { ...all[idx], ...patch, updatedAt: nowIso() };
  setAll(all);
  return ok({ id });
}

export async function deleteCheckupApi(id) {
  if (useRemoteApi()) {
    return apiDelete(`/api/v1/checkups/${encodeURIComponent(id)}`);
  }
  const all = getAll();
  const next = all.filter((x) => x.id !== id);
  if (next.length === all.length) return fail("产检记录不存在", "E_NOT_FOUND", {});
  setAll(next);
  return ok({});
}

export async function addCheckupReportsApi(id, { images = [], summary = "", note = "" } = {}) {
  if (!Array.isArray(images) || images.length === 0) return fail("请选择要上传的图片", "E_PARAM_INVALID", {});
  if (images.length > 9) return fail("最多上传9张图片", "E_PARAM_INVALID", {});

  if (useRemoteApi()) {
    try {
      const last = await uploadReportsSequential(id, images, summary, note);
      if (!last || last.code !== 0) {
        return fail(last?.message || "上传失败", last?.errorCode || "E_NETWORK", last?.data || {}, last?.code);
      }
      return last;
    } catch (e) {
      return fail(String(e?.errMsg || e?.message || e), "E_NETWORK", {});
    }
  }

  const all = getAll();
  const idx = all.findIndex((x) => x.id === id);
  if (idx < 0) return fail("产检记录不存在", "E_NOT_FOUND", {});

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
