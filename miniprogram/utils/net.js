let cachedOnline = true;

export function isOnline() {
  return cachedOnline;
}

export function initNetworkListener(onChange) {
  try {
    wx.getNetworkType({
      success(res) {
        cachedOnline = res.networkType !== "none";
        if (typeof onChange === "function") onChange(cachedOnline, res.networkType);
      }
    });
  } catch (e) {}

  try {
    wx.onNetworkStatusChange((res) => {
      cachedOnline = !!res.isConnected;
      if (typeof onChange === "function") onChange(cachedOnline, res.networkType);
    });
  } catch (e) {}
}

