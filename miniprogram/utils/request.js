import { fail } from "./errors";

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

export async function request({
  url,
  method = "GET",
  header = {},
  data,
  timeout = 8000,
  retries = 2
} = {}) {
  if (!url) return fail("缺少 url", "E_PARAM_INVALID", {});

  let lastErr = null;
  for (let i = 0; i <= retries; i += 1) {
    try {
      const res = await new Promise((resolve, reject) => {
        wx.request({
          url,
          method,
          header,
          data,
          timeout,
          success: resolve,
          fail: reject
        });
      });

      const body = res?.data;
      if (body && typeof body === "object" && typeof body.code === "number") return body;
      // 非标准响应：包一层
      return { code: 0, message: "success", data: body ?? {} };
    } catch (e) {
      lastErr = e;
      if (i < retries) {
        await sleep(250 * (i + 1));
        continue;
      }
    }
  }

  return fail("网络不稳定，请稍后重试", "E_NETWORK", { raw: String(lastErr?.errMsg || lastErr?.message || lastErr) });
}

