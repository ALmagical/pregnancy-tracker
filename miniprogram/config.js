/**
 * 后端部署在腾讯云服务器时，将此处改为你的 HTTPS 根地址（无尾斜杠）。
 * 小程序「开发管理 → 开发设置 → 服务器域名」需添加相同 request 合法域名。
 *
 * 留空字符串则使用本地存储逻辑（离线演示模式）。
 */
export const API_BASE_URL = "";

/**
 * 微信 wx.login 返回的 code 换取 JWT / session token 的接口路径。
 * 请求体：{ "code": "..." }
 * 响应 data 含 token 或 accessToken 字段即可被 app 识别。
 */
export const AUTH_WECHAT_LOGIN_PATH = "/api/v1/auth/wechat";

/**
 * 胎动历史列表（设计文档 6.6.4 仅有按日汇总）。若你的后端提供会话列表，保持默认路径；
 * 否则可在服务端实现 GET /api/v1/fetal-movements/sessions 或改此常量。
 */
export const FM_SESSIONS_LIST_PATH = "/api/v1/fetal-movements/sessions";

export function useRemoteApi() {
  const b = String(API_BASE_URL || "").trim();
  return /^https:\/\//i.test(b);
}
