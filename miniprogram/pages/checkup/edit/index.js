import { createCheckupApi, getCheckupDetailApi, updateCheckupApi } from "../../../services/checkupService";
import { mapErrorToUserMessage } from "../../../utils/errors";
import { track } from "../../../utils/track";

Page({
  data: {
    online: true,
    offlineQueueSize: 0,
    id: "",
    checkupDate: "",
    checkupType: "",
    hospital: "",
    note: ""
  },
  async onLoad(query) {
    const app = getApp();
    this.setData({
      online: app?.globalData?.online !== false,
      offlineQueueSize: app?.globalData?.offlineQueueSize || 0,
      id: query?.id || ""
    });
    if (query?.id) await this.loadDetail(query.id);
  },
  async loadDetail(id) {
    try {
      const res = await getCheckupDetailApi(id);
      if (res.code !== 0) throw res;
      const c = res.data;
      this.setData({
        checkupDate: c.checkupDate || "",
        checkupType: c.checkupType || "",
        hospital: c.hospital || "",
        note: c.note || ""
      });
    } catch (e) {
      wx.showToast({ title: mapErrorToUserMessage(e), icon: "none" });
    }
  },
  onDateChange(e) {
    this.setData({ checkupDate: e.detail.value });
  },
  onTypeInput(e) {
    this.setData({ checkupType: e.detail.value });
  },
  onHospitalInput(e) {
    this.setData({ hospital: e.detail.value });
  },
  onNoteInput(e) {
    this.setData({ note: e.detail.value });
  },
  async onSave() {
    try {
      const payload = {
        checkupDate: this.data.checkupDate,
        checkupType: (this.data.checkupType || "").trim(),
        hospital: (this.data.hospital || "").trim(),
        note: (this.data.note || "").trim()
      };
      if (this.data.id) {
        const res = await updateCheckupApi(this.data.id, payload);
        if (res.code !== 0) throw res;
      } else {
        const res = await createCheckupApi(payload);
        if (res.code !== 0) throw res;
        track("checkup_add_submit", {
          checkup_type_id: payload.checkupTypeId || "manual",
          days_to_checkup: "unknown"
        });
      }
      wx.showToast({ title: "已保存", icon: "success" });
      setTimeout(() => wx.navigateBack({ delta: 1 }), 400);
    } catch (e) {
      wx.showToast({ title: mapErrorToUserMessage(e), icon: "none" });
    }
  }
});

