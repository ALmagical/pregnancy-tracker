Page({
  data: {
    online: true,
    offlineQueueSize: 0,
    tools: [
      {
        key: "weight",
        title: "体重",
        sub: "记录与趋势",
        icon: "/assets/icons/weight.svg",
        action: "goWeight"
      },
      {
        key: "fm",
        title: "胎动",
        sub: "开始计数",
        icon: "/assets/icons/baby.svg",
        action: "goFm"
      },
      {
        key: "contraction",
        title: "宫缩",
        sub: "计时与记录",
        icon: "/assets/icons/timer.svg",
        action: "goContraction"
      },
      {
        key: "checklist",
        title: "待产清单",
        sub: "模板 + 自定义",
        icon: "/assets/icons/clipboard.svg",
        action: "goChecklist"
      }
    ]
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
  },
  onToolTap(e) {
    const action = e.currentTarget?.dataset?.action;
    if (action && typeof this[action] === "function") {
      this[action]();
    }
  }
});

