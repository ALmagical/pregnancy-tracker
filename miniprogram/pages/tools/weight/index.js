import { listWeightsApi, createWeightApi, deleteWeightApi } from "../../../services/weightService";
import { mapErrorToUserMessage } from "../../../utils/errors";
import { track } from "../../../utils/track";

Page({
  data: {
    state: "loading",
    list: [],
    stats: { currentWeight: "-" },
    online: true,
    offlineQueueSize: 0
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
    this.setData({ state: "loading" });
    try {
      const res = await listWeightsApi({ page: 1, pageSize: 30 });
      if (res.code !== 0) throw res;
      const list = res.data.list || [];
      if (!list.length) {
        this.setData({ state: "empty", list: [], stats: { currentWeight: "-" } });
        return;
      }
      this.setData({ state: "normal", list, stats: res.data.statistics || { currentWeight: "-" } });
    } catch (e) {
      this.setData({ state: "error" });
      wx.showToast({ title: mapErrorToUserMessage(e), icon: "none" });
    }
  },
  async onAdd() {
    // 轻量实现：用系统输入框替代复杂表单
    wx.showModal({
      title: "新增体重",
      editable: true,
      placeholderText: "例如 62.3",
      success: async (r) => {
        if (!r.confirm) return;
        const weightStr = (r.content || "").trim();
        const res = await createWeightApi({ weight: weightStr });
        if (res.code !== 0) {
          if (res.errorCode === "E_WEIGHT_UNREASONABLE") {
            wx.showModal({
              title: "确认保存？",
              content: "体重似乎超出常见范围，是否仍要保存？",
              success: async (rr) => {
                if (!rr.confirm) return;
                await createWeightApi({ weight: weightStr });
                track("weight_add_success", { source: "manual", is_outlier_confirmed: true });
                await this.reload();
              }
            });
            return;
          }
          wx.showToast({ title: res.message || "保存失败", icon: "none" });
          return;
        }
        track("weight_add_success", { source: "manual", is_outlier_confirmed: false });
        if (!this.data.online) {
          wx.showToast({ title: "已暂存，联网后自动同步", icon: "none" });
        }
        await this.reload();
      }
    });
  },
  async onDelete(e) {
    const id = e.currentTarget?.dataset?.id;
    if (!id) return;
    wx.showModal({
      title: "确认删除？",
      content: "删除后不可恢复",
      success: async (r) => {
        if (!r.confirm) return;
        const res = await deleteWeightApi(id);
        if (res.code !== 0) {
          wx.showToast({ title: res.message || "删除失败", icon: "none" });
          return;
        }
        await this.reload();
      }
    });
  }
});

