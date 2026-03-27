import { createContractionApi, listContractionsApi } from "../../../services/contractionService";
import { track } from "../../../utils/track";

function formatElapsed(ms) {
  const s = Math.floor(ms / 1000);
  const mm = String(Math.floor(s / 60)).padStart(2, "0");
  const ss = String(s % 60).padStart(2, "0");
  return `${mm}:${ss}`;
}

Page({
  data: {
    online: true,
    offlineQueueSize: 0,
    running: false,
    startedAt: null,
    elapsedText: "00:00",
    list: []
  },
  timer: null,
  async onShow() {
    const app = getApp();
    this.setData({
      online: app?.globalData?.online !== false,
      offlineQueueSize: app?.globalData?.offlineQueueSize || 0
    });
    await this.reload();
  },
  onHide() {
    this.clearTimer();
  },
  clearTimer() {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
  },
  async reload() {
    const res = await listContractionsApi();
    this.setData({ list: res.data.list || [] });
  },
  onStart() {
    const startedAt = new Date().toISOString();
    this.setData({ running: true, startedAt, elapsedText: "00:00" });
    const startMs = Date.now();
    this.clearTimer();
    this.timer = setInterval(() => {
      this.setData({ elapsedText: formatElapsed(Date.now() - startMs) });
    }, 500);
  },
  async onStop() {
    const endedAt = new Date().toISOString();
    const startedAt = this.data.startedAt;
    this.clearTimer();
    this.setData({ running: false, startedAt: null, elapsedText: "00:00" });
    const res = await createContractionApi({ startedAt, endedAt });
    if (res.code !== 0) {
      wx.showToast({ title: res.message || "保存失败", icon: "none" });
      return;
    }
    const durationSec = Math.floor((Date.parse(endedAt) - Date.parse(startedAt)) / 1000);
    track("contraction_add_success", { duration_sec: durationSec, interval_sec_bucket: "unknown" });
    if (!this.data.online) {
      wx.showToast({ title: "已暂存，联网后自动同步", icon: "none" });
    }
    await this.reload();
  }
});

