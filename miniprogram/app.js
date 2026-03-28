import { API_BASE_URL, useRemoteApi } from "./config";
import { ensureWechatSession } from "./services/authService";
import { initMockDbIfNeeded } from "./services/mockDb";
import { initNetworkListener } from "./utils/net";
import { getOfflineQueueSize } from "./services/offlineSync";
import { syncOfflineQueue } from "./services/offlineSync";

App({
  globalData: {
    env: "dev",
    online: true,
    offlineQueueSize: 0,
    apiBase: "",
    useRemoteApi: false
  },
  async onLaunch() {
    this.globalData.apiBase = API_BASE_URL;
    this.globalData.useRemoteApi = useRemoteApi();
    if (useRemoteApi()) {
      await ensureWechatSession();
    }
    initMockDbIfNeeded();

    initNetworkListener(async (online) => {
      this.globalData.online = online;
      this.globalData.offlineQueueSize = getOfflineQueueSize();
      if (online) {
        const before = this.globalData.offlineQueueSize || 0;
        const res = await syncOfflineQueue();
        this.globalData.offlineQueueSize = getOfflineQueueSize();
        const after = this.globalData.offlineQueueSize || 0;
        const processed = typeof res?.processed === "number" ? res.processed : Math.max(0, before - after);
        if (processed > 0) {
          wx.showToast({ title: `已同步${processed}条暂存记录`, icon: "none" });
        }
      }
    });
  }
});

