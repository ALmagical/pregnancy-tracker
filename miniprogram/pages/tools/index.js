Page({
  data: {
    online: true,
    offlineQueueSize: 0
  },
  onShow() {
    const app = getApp();
    this.setData({
      online: app?.globalData?.online !== false,
      offlineQueueSize: app?.globalData?.offlineQueueSize || 0
    });
  },
  goWeight() {
    wx.navigateTo({ url: "/pages/tools/weight/index" });
  },
  goFm() {
    wx.navigateTo({ url: "/pages/tools/fm/index" });
  },
  goContraction() {
    wx.navigateTo({ url: "/pages/tools/contraction/index" });
  },
  goChecklist() {
    wx.navigateTo({ url: "/pages/tools/checklist/index" });
  }
});

