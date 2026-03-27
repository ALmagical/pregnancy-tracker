import { dequeueOfflineActions, getOfflineQueue, setOfflineQueue } from "../utils/storage";
import { isOnline } from "../utils/net";

let syncing = false;

export function getOfflineQueueSize() {
  return getOfflineQueue().length;
}

export async function syncOfflineQueue({ batchSize = 20 } = {}) {
  if (syncing) return { ok: true, skipped: true };
  if (!isOnline()) return { ok: false, error: "offline" };
  syncing = true;
  try {
    // 当前阶段没有真实后端：用“清空队列”模拟同步成功
    // 后续接入真实后端时，在此处按 action.type 分发调用网络 API
    const actions = dequeueOfflineActions(batchSize);
    if (!actions.length) return { ok: true, processed: 0 };

    // 模拟网络耗时
    await new Promise((r) => setTimeout(r, 120));

    // 如果还有剩余队列，递归继续
    const remain = getOfflineQueue().length;
    if (remain > 0) {
      // 让出事件循环避免卡顿
      await new Promise((r) => setTimeout(r, 0));
      return await syncOfflineQueue({ batchSize });
    }
    return { ok: true, processed: actions.length };
  } catch (e) {
    // 失败则将本批次塞回队列头部（保守处理）
    const current = getOfflineQueue();
    setOfflineQueue(current);
    return { ok: false, error: String(e?.message || e) };
  } finally {
    syncing = false;
  }
}

