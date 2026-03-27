import { getUserInfo, getSettings } from "./storage";
import { isOnline } from "./net";

function bucketLen(n) {
  if (!Number.isFinite(n)) return "unknown";
  if (n <= 20) return "0_20";
  if (n <= 50) return "21_50";
  if (n <= 100) return "51_100";
  if (n <= 200) return "101_200";
  return "200_plus";
}

function sanitizeProps(props) {
  const p = { ...(props || {}) };
  // 禁止上报自由文本原文：统一丢弃 note/summary/question 等字段
  const forbiddenKeys = ["note", "summary", "question", "content", "hospital", "dueDate", "lastPeriodDate"];
  forbiddenKeys.forEach((k) => {
    if (typeof p[k] !== "undefined") delete p[k];
  });
  // 如传入 msg_len，改成分桶
  if (typeof p.msg_len === "number") {
    p.msg_len_bucket = bucketLen(p.msg_len);
    delete p.msg_len;
  }
  return p;
}

export function track(eventName, props) {
  const userInfo = getUserInfo();
  const settings = getSettings();
  const common = {
    env: getApp()?.globalData?.env || "dev",
    network: isOnline() ? "online" : "none",
    ts: new Date().toISOString(),
    user_status: userInfo?.status || "unknown",
    week: userInfo?.currentWeek ?? null,
    day: userInfo?.currentDay ?? null,
    ai_context_enabled: settings?.ai?.contextEnabled !== false
  };

  const payload = {
    event: eventName,
    ...common,
    props: sanitizeProps(props)
  };

  // 当前阶段：仅 console 输出；后续可接入上报服务
  // eslint-disable-next-line no-console
  console.log("[track]", payload);
}

