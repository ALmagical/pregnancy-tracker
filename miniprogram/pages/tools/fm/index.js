import {
  getActiveFmSessionApi,
  startFmSessionApi,
  incFmCountApi,
  finishFmSessionApi,
  listFmSessionsApi
} from "../../../services/fmService";
import { track } from "../../../utils/track";

Page({
  data: {
    online: true,
    offlineQueueSize: 0,
    active: null,
    sessions: []
  },
  async onShow() {
    const app = getApp();
    this.setData({
      online: app?.globalData?.online !== false,
      offlineQueueSize: app?.globalData?.offlineQueueSize || 0
    });
    await this.reload();
  },
  async reload() {
    const a = await getActiveFmSessionApi();
    const s = await listFmSessionsApi();
    this.setData({ active: a.data.active, sessions: s.data.list || [] });
  },
  async onStart() {
    const res = await startFmSessionApi();
    if (res.code !== 0) {
      wx.showToast({ title: res.message || "无法开始", icon: "none" });
      return;
    }
    await this.reload();
  },
  async onPlus() {
    const res = await incFmCountApi(1);
    if (res.code !== 0) {
      wx.showToast({ title: res.message || "操作失败", icon: "none" });
      return;
    }
    this.setData({ active: res.data.active });
  },
  async onMinus() {
    const res = await incFmCountApi(-1);
    if (res.code !== 0) {
      wx.showToast({ title: res.message || "操作失败", icon: "none" });
      return;
    }
    this.setData({ active: res.data.active });
  },
  async onFinish() {
    const res = await finishFmSessionApi();
    if (res.code !== 0) {
      wx.showToast({ title: res.message || "保存失败", icon: "none" });
      return;
    }
    track("fm_session_finish", {
      count: res.data.session.count,
      duration_sec: res.data.session.durationSec,
      result_tag: "saved"
    });
    if (!this.data.online) {
      wx.showToast({ title: "已暂存，联网后自动同步", icon: "none" });
    }
    await this.reload();
  }
});

