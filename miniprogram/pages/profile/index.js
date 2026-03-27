import { getUserInfoApi, updateUserInfoApi } from "../../services/userService";
import { mapErrorToUserMessage } from "../../utils/errors";
import { track } from "../../utils/track";

const STATUS_OPTIONS = [
  { value: "trying", label: "备孕" },
  { value: "pregnant", label: "孕期" },
  { value: "postpartum", label: "产后" }
];

Page({
  data: {
    online: true,
    offlineQueueSize: 0,
    statusOptions: STATUS_OPTIONS,
    statusIndex: 1,
    lastPeriodDate: "",
    heightCm: "",
    prePregnancyWeight: ""
  },
  async onShow() {
    const app = getApp();
    this.setData({
      online: app?.globalData?.online !== false,
      offlineQueueSize: app?.globalData?.offlineQueueSize || 0
    });
    await this.load();
  },
  async load() {
    const res = await getUserInfoApi();
    if (res.code !== 0) return;
    const u = res.data;
    const statusIndex = Math.max(
      0,
      STATUS_OPTIONS.findIndex((x) => x.value === (u.status || "pregnant"))
    );
    this.setData({
      statusIndex,
      lastPeriodDate: u.lastPeriodDate || "",
      heightCm: u.heightCm ? String(u.heightCm) : "",
      prePregnancyWeight: u.prePregnancyWeight != null ? String(u.prePregnancyWeight) : ""
    });
  },
  onStatusChange(e) {
    this.setData({ statusIndex: Number(e.detail.value) || 0 });
  },
  onDateChange(e) {
    this.setData({ lastPeriodDate: e.detail.value });
  },
  onHeightInput(e) {
    this.setData({ heightCm: e.detail.value });
  },
  onPreWeightInput(e) {
    this.setData({ prePregnancyWeight: e.detail.value });
  },
  async onSave() {
    try {
      const status = STATUS_OPTIONS[this.data.statusIndex]?.value || "pregnant";
      const payload = {
        status,
        lastPeriodDate: this.data.lastPeriodDate,
        heightCm: this.data.heightCm ? Number(this.data.heightCm) : undefined,
        prePregnancyWeight: this.data.prePregnancyWeight ? Number(this.data.prePregnancyWeight) : undefined
      };
      const res = await updateUserInfoApi(payload);
      if (res.code !== 0) throw res;
      track("profile_edit_success", {});
      wx.showToast({ title: "已保存", icon: "success" });
      setTimeout(() => wx.navigateBack({ delta: 1 }), 400);
    } catch (e) {
      wx.showToast({ title: mapErrorToUserMessage(e), icon: "none" });
    }
  }
});

