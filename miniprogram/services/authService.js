import { AUTH_WECHAT_LOGIN_PATH, useRemoteApi } from "../config";
import { buildApiUrl } from "../utils/apiClient";
import { request } from "../utils/request";
import { setAuthToken } from "../utils/storage";

/**
 * 远程模式下在启动时调用：wx.login → 后端换 token。
 * 后端字段兼容：data.token 或 data.accessToken
 */
export function ensureWechatSession() {
  if (!useRemoteApi()) return Promise.resolve(true);

  return new Promise((resolve) => {
    wx.login({
      success: async (res) => {
        if (!res.code) {
          resolve(false);
          return;
        }
        try {
          const raw = await request({
            url: buildApiUrl(AUTH_WECHAT_LOGIN_PATH),
            method: "POST",
            header: { "Content-Type": "application/json" },
            data: { code: res.code }
          });
          if (raw && raw.code === 0 && raw.data) {
            const t = raw.data.accessToken || raw.data.token;
            if (t) {
              setAuthToken(String(t));
              resolve(true);
              return;
            }
          }
        } catch (e) {}
        resolve(false);
      },
      fail: () => resolve(false)
    });
  });
}
