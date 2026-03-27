import { getUserInfoApi } from "../../services/userService";
import { listCheckupsApi } from "../../services/checkupService";
import { diffDays, parseYYYYMMDD } from "../../utils/date";
import { mapErrorToUserMessage } from "../../utils/errors";
import { track } from "../../utils/track";

Page({
  data: {
    online: true,
    offlineQueueSize: 0,
    state: "loading",
    userInfo: null,
    daysToDue: "-",
    nextCheckupText: "暂无"
  },
  async onShow() {
    const app = getApp();
    this.setData({
      online: app?.globalData?.online !== false,
      offlineQueueSize: app?.globalData?.offlineQueueSize || 0
    });
    track("home_view", { entry_source: "tab" });
    await this.reload();
  },
  async reload() {
    this.setData({ state: "loading" });
    try {
      const u = await getUserInfoApi();
      if (u.code !== 0) {
        this.setData({ state: "empty", userInfo: null });
        return;
      }
      const userInfo = u.data;
      const due = parseYYYYMMDD(userInfo.dueDate);
      const daysToDue = due ? Math.max(0, -diffDays(new Date(), due)) : "-";

      const c = await listCheckupsApi({ page: 1, pageSize: 20 });
      let nextText = "暂无";
      if (c.code === 0) {
        const today = new Date();
        const upcoming = (c.data.list || [])
          .filter((x) => x.checkupDate)
          .map((x) => ({ ...x, d: parseYYYYMMDD(x.checkupDate) }))
          .filter((x) => x.d && diffDays(x.d, today) >= 0)
          .sort((a, b) => a.d.getTime() - b.d.getTime())[0];
        if (upcoming) {
          const delta = diffDays(upcoming.d, today);
          nextText = `${upcoming.checkupDate} · ${upcoming.checkupType || "产检"} · ${delta}天后`;
        }
      }

      this.setData({ state: "normal", userInfo, daysToDue, nextCheckupText: nextText });
    } catch (e) {
      this.setData({ state: "error" });
      wx.showToast({ title: mapErrorToUserMessage(e), icon: "none" });
    }
  },
  goProfile() {
    wx.navigateTo({ url: "/pages/profile/index" });
  },
  goCheckup() {
    wx.switchTab({ url: "/pages/checkup/index" });
  },
  onQuickTool(e) {
    const tool = e.currentTarget?.dataset?.tool;
    if (!tool) return;
    track("home_quick_tool_click", { tool });
    const map = {
      weight: "/pages/tools/weight/index",
      fm: "/pages/tools/fm/index",
      contraction: "/pages/tools/contraction/index",
      checklist: "/pages/tools/checklist/index"
    };
    wx.navigateTo({ url: map[tool] || "/pages/tools/index" });
  }
});

