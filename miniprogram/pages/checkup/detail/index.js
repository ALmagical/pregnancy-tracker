import { addCheckupReportsApi, getCheckupDetailApi } from "../../../services/checkupService";
import { isOnline } from "../../../utils/net";
import { mapErrorToUserMessage } from "../../../utils/errors";
import { track } from "../../../utils/track";

Page({
  data: {
    online: true,
    offlineQueueSize: 0,
    id: "",
    state: "loading",
    detail: null,
    checkupTypeText: "",
    checkupMetaText: "",
    hasImages: false,
    images: [],
    summaryText: "未填写",
    noteText: "未填写",
    uploading: false
  },
  async onLoad(query) {
    const app = getApp();
    this.setData({
      online: !(app && app.globalData && app.globalData.online === false),
      offlineQueueSize:
        (app && app.globalData && app.globalData.offlineQueueSize) ? app.globalData.offlineQueueSize : 0,
      id: (query && query.id) ? query.id : ""
    });
    await this.reload();
  },
  async reload() {
    this.setData({ state: "loading" });
    try {
      const res = await getCheckupDetailApi(this.data.id);
      if (res.code !== 0) throw res;
      const detail = res.data || {};
      const checkupTypeText = detail.checkupType ? String(detail.checkupType) : "产检";
      const checkupMetaText = `${detail.checkupDate || ""} · ${detail.hospital ? String(detail.hospital) : "未填写医院"}`;
      const rawImages = Array.isArray(detail.images) ? detail.images : [];
      const images = rawImages.map((x) => ({
        id: x.id,
        url: x.url,
        thumb: x.thumbnail || x.url
      }));
      const hasImages = images.length > 0;
      const summaryText = detail.summary ? String(detail.summary) : "未填写";
      const noteText = detail.note ? String(detail.note) : "未填写";
      this.setData({
        state: "normal",
        detail,
        checkupTypeText,
        checkupMetaText,
        hasImages,
        images,
        summaryText,
        noteText
      });
    } catch (e) {
      this.setData({ state: "error" });
      wx.showToast({ title: mapErrorToUserMessage(e), icon: "none" });
    }
  },
  onPreview(e) {
    const idx =
      Number(e && e.currentTarget && e.currentTarget.dataset && e.currentTarget.dataset.idx) || 0;
    const urls = (this.data.images || []).map((x) => x.url);
    if (!urls.length) return;
    wx.previewImage({ urls, current: urls[idx] });
  },
  async onUpload() {
    if (!isOnline()) {
      wx.showToast({ title: "离线状态下无法上传报告图片", icon: "none" });
      return;
    }
    if (this.data.uploading) return;

    try {
      const choose = await new Promise((resolve, reject) => {
        wx.chooseImage({
          count: 9,
          sizeType: ["compressed", "original"],
          sourceType: ["album", "camera"],
          success: resolve,
          fail: reject
        });
      });
      const paths = (choose && choose.tempFilePaths) ? choose.tempFilePaths : [];
      if (!paths.length) return;
      if (paths.length > 9) {
        wx.showToast({ title: "最多选择9张图片", icon: "none" });
        return;
      }
      this.setData({ uploading: true });
      const res = await addCheckupReportsApi(this.data.id, { images: paths });
      if (res.code !== 0) throw res;
      track("checkup_report_upload_success", { image_count: paths.length, report_type: "image" });
      await this.reload();
    } catch (e) {
      wx.showToast({ title: mapErrorToUserMessage(e), icon: "none" });
    } finally {
      this.setData({ uploading: false });
    }
  },
  onAiInterpret() {
    track("report_ai_interpret_click", {
      checkup_type_id: (this.data.detail && this.data.detail.checkupTypeId) ? this.data.detail.checkupTypeId : "manual"
    });
    wx.showToast({ title: "AI 解读将在后续版本提供", icon: "none" });
  }
});

