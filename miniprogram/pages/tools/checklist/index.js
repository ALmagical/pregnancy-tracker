import { addChecklistItemApi, getChecklistApi, resetChecklistApi, toggleChecklistItemApi } from "../../../services/checklistService";
import { track } from "../../../utils/track";

Page({
  data: {
    online: true,
    offlineQueueSize: 0,
    state: "loading",
    items: [],
    checkedCount: 0
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
    const res = await getChecklistApi();
    const items = res.data.items || [];
    if (!items.length) {
      this.setData({ state: "empty", items: [], checkedCount: 0 });
      return;
    }
    const checkedCount = items.filter((x) => x.checked).length;
    this.setData({ state: "normal", items, checkedCount });
  },
  async onToggle(e) {
    const id = e.currentTarget?.dataset?.id;
    if (!id) return;
    const item = this.data.items.find((x) => x.id === id);
    if (!item) return;
    const res = await toggleChecklistItemApi(id, !item.checked);
    if (res.code !== 0) {
      wx.showToast({ title: res.message || "操作失败", icon: "none" });
      return;
    }
    track("checklist_item_toggle", {
      item_source: item.source || "unknown",
      category_id: item.categoryId || "unknown"
    });
    if (!this.data.online) {
      wx.showToast({ title: "已暂存，联网后自动同步", icon: "none" });
    }
    await this.reload();
  },
  onAdd() {
    wx.showModal({
      title: "新增清单项",
      editable: true,
      placeholderText: "例如：一次性马桶垫",
      success: async (r) => {
        if (!r.confirm) return;
        const title = (r.content || "").trim();
        const res = await addChecklistItemApi({ title });
        if (res.code !== 0) {
          wx.showToast({ title: res.message || "新增失败", icon: "none" });
          return;
        }
        if (!this.data.online) {
          wx.showToast({ title: "已暂存，联网后自动同步", icon: "none" });
        }
        await this.reload();
      }
    });
  },
  onReset() {
    wx.showModal({
      title: "恢复默认？",
      content: "将清空自定义项并重置勾选状态",
      success: async (r) => {
        if (!r.confirm) return;
        const res = await resetChecklistApi();
        if (res.code !== 0) {
          wx.showToast({ title: res.message || "操作失败", icon: "none" });
          return;
        }
        if (!this.data.online) {
          wx.showToast({ title: "已暂存，联网后自动同步", icon: "none" });
        }
        await this.reload();
      }
    });
  }
});

