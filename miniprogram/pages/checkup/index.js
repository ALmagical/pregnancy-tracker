import { deleteCheckupApi, listCheckupsApi } from "../../services/checkupService";
import { mapErrorToUserMessage } from "../../utils/errors";

Page({
  data: {
    online: true,
    offlineQueueSize: 0,
    state: "loading",
    list: []
  },
  async onShow() {
    const app = getApp();
    this.setData({
      online: app?.globalData?.online !== false,
      offlineQueueSize: app?.globalData?.offlineQueueSize || 0
    });
    await this.reload();
  },
  noop() {},
  async reload() {
    this.setData({ state: "loading" });
    try {
      const res = await listCheckupsApi({ page: 1, pageSize: 50 });
      if (res.code !== 0) throw res;
      const list = res.data.list || [];
      if (!list.length) {
        this.setData({ state: "empty", list: [] });
        return;
      }
      this.setData({ state: "normal", list });
    } catch (e) {
      this.setData({ state: "error" });
      wx.showToast({ title: mapErrorToUserMessage(e), icon: "none" });
    }
  },
  onAdd() {
    wx.navigateTo({ url: "/pages/checkup/edit/index" });
  },
  onEdit(e) {
    const id = e.currentTarget?.dataset?.id;
    wx.navigateTo({ url: `/pages/checkup/edit/index?id=${encodeURIComponent(id)}` });
  },
  onOpenDetail(e) {
    const id = e.currentTarget?.dataset?.id;
    if (!id) return;
    wx.navigateTo({ url: `/pages/checkup/detail/index?id=${encodeURIComponent(id)}` });
  },
  onDelete(e) {
    const id = e.currentTarget?.dataset?.id;
    if (!id) return;
    wx.showModal({
      title: "确认删除？",
      content: "删除后不可恢复",
      success: async (r) => {
        if (!r.confirm) return;
        const res = await deleteCheckupApi(id);
        if (res.code !== 0) {
          wx.showToast({ title: res.message || "删除失败", icon: "none" });
          return;
        }
        await this.reload();
      }
    });
  }
});

