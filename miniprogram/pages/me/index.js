import { getUserInfoApi } from "../../services/userService";
import { getSettings, setSettings } from "../../utils/storage";
import { mapErrorToUserMessage } from "../../utils/errors";

Page({
  data: {
    online: true,
    offlineQueueSize: 0,
    state: "loading",
    userInfo: null,
    aiContextEnabled: true
  },
  async onShow() {
    const app = getApp();
    const settings = getSettings();
    this.setData({
      online: app?.globalData?.online !== false,
      offlineQueueSize: app?.globalData?.offlineQueueSize || 0,
      aiContextEnabled: settings?.ai?.contextEnabled !== false
    });
    await this.reload();
  },
  async reload() {
    this.setData({ state: "loading" });
    try {
      const res = await getUserInfoApi();
      if (res.code !== 0) {
        this.setData({ state: "empty", userInfo: null });
        return;
      }
      this.setData({ state: "normal", userInfo: res.data });
    } catch (e) {
      this.setData({ state: "error" });
      wx.showToast({ title: mapErrorToUserMessage(e), icon: "none" });
    }
  },
  goProfile() {
    wx.navigateTo({ url: "/pages/profile/index" });
  },
  onAiContextToggle(e) {
    const enabled = !!e.detail.value;
    const settings = getSettings();
    const next = { ...(settings || {}), ai: { ...(settings?.ai || {}), contextEnabled: enabled } };
    setSettings(next);
    this.setData({ aiContextEnabled: enabled });
    wx.showToast({ title: enabled ? "已开启" : "已关闭", icon: "none" });
  },
  onExport() {
    wx.showToast({ title: "导出功能将在后续里程碑提供", icon: "none" });
  }
});

