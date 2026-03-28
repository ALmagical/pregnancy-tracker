import { useRemoteApi } from "../config";
import { apiDelete, apiPost, apiPut } from "../utils/apiClient";
import { dequeueOfflineActions, getOfflineQueue, setOfflineQueue } from "../utils/storage";
import { isOnline } from "../utils/net";

let syncing = false;

export function getOfflineQueueSize() {
  return getOfflineQueue().length;
}

async function replayRemoteAction(action) {
  const p = action?.payload;
  switch (action?.type) {
    case "weight.create": {
      const r = await apiPost("/api/v1/weights", {
        weight: p.weight,
        recordedAt: String(p.recordedAt || "").slice(0, 10),
        note: ""
      });
      return r.code === 0;
    }
    case "weight.delete": {
      const r = await apiDelete(`/api/v1/weights/${encodeURIComponent(p.id)}`);
      return r.code === 0;
    }
    case "contraction.create": {
      const r = await apiPost("/api/v1/contractions", { startedAt: p.startedAt, endedAt: p.endedAt });
      return r.code === 0;
    }
    case "checklist.toggle": {
      const r = await apiPut(`/api/v1/checklist/items/${encodeURIComponent(p.id)}`, {
        checked: !!p.checked,
        note: ""
      });
      return r.code === 0;
    }
    case "checklist.add": {
      const r = await apiPost("/api/v1/checklist/items", {
        categoryId: p.categoryId || "custom",
        title: p.title,
        note: p.note || ""
      });
      return r.code === 0;
    }
    case "checklist.reset": {
      const r = await apiPost("/api/v1/checklist/reset", { keepCustomItems: false });
      return r.code === 0;
    }
    case "fm.finish":
      return true;
    default:
      return true;
  }
}

export async function syncOfflineQueue({ batchSize = 20 } = {}) {
  if (syncing) return { ok: true, skipped: true };
  if (!isOnline()) return { ok: false, error: "offline" };
  syncing = true;
  try {
    const q = getOfflineQueue();
    if (!q.length) return { ok: true, processed: 0 };

    if (!useRemoteApi()) {
      const actions = dequeueOfflineActions(batchSize);
      if (!actions.length) return { ok: true, processed: 0 };
      await new Promise((r) => setTimeout(r, 120));
      const remain = getOfflineQueue().length;
      if (remain > 0) {
        await new Promise((r) => setTimeout(r, 0));
        const next = await syncOfflineQueue({ batchSize });
        return { ok: true, processed: actions.length + (next.processed || 0) };
      }
      return { ok: true, processed: actions.length };
    }

    let processed = 0;
    let idx = 0;
    while (idx < q.length && processed < batchSize) {
      const okReplay = await replayRemoteAction(q[idx]);
      if (!okReplay) break;
      idx += 1;
      processed += 1;
    }
    setOfflineQueue(q.slice(idx));

    if (idx < q.length && processed > 0) {
      await new Promise((r) => setTimeout(r, 0));
      const next = await syncOfflineQueue({ batchSize });
      return { ok: true, processed: processed + (next.processed || 0) };
    }

    return { ok: true, processed };
  } catch (e) {
    return { ok: false, error: String(e?.message || e) };
  } finally {
    syncing = false;
  }
}
