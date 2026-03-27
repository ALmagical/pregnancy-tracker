export function ok(data) {
  return { code: 0, message: "success", data: data ?? {} };
}

export function fail(message, errorCode, data, code) {
  return {
    code: typeof code === "number" ? code : 10001,
    message: message || "请求失败",
    errorCode: errorCode || "E_NETWORK",
    data: data || {}
  };
}

export function mapErrorToUserMessage(err) {
  if (!err) return "请求失败，请稍后重试";
  if (typeof err === "string") return err;
  const message = err.message || err.msg || "";
  const errorCode = err.errorCode || "";
  if (message) return message;
  switch (errorCode) {
    case "E_UNAUTHORIZED":
      return "登录已过期，请重新授权";
    case "E_RATE_LIMITED":
      return "操作太频繁了，请稍后再试";
    case "E_NETWORK":
      return "网络异常，请检查后重试";
    default:
      return "请求失败，请稍后重试";
  }
}

