import { STORAGE_KEYS } from "./constants";

function get(key, fallback) {
  try {
    const v = wx.getStorageSync(key);
    return v === "" || typeof v === "undefined" ? fallback : v;
  } catch (e) {
    return fallback;
  }
}

function set(key, value) {
  wx.setStorageSync(key, value);
}

export const Storage = {
  get,
  set,
  remove(key) {
    try {
      wx.removeStorageSync(key);
    } catch (e) {}
  }
};

export function getUserInfo() {
  return get(STORAGE_KEYS.userInfo, null);
}

export function setUserInfo(userInfo) {
  set(STORAGE_KEYS.userInfo, userInfo);
}

export function getSettings() {
  return get(STORAGE_KEYS.settings, { ai: { contextEnabled: true } });
}

export function setSettings(settings) {
  set(STORAGE_KEYS.settings, settings);
}

export function getOfflineQueue() {
  return get(STORAGE_KEYS.offlineQueue, []);
}

export function setOfflineQueue(queue) {
  set(STORAGE_KEYS.offlineQueue, queue);
}

export function enqueueOfflineAction(action) {
  const q = getOfflineQueue();
  q.push({ ...action, enqueuedAt: Date.now() });
  setOfflineQueue(q);
  try {
    const app = getApp();
    if (app?.globalData) app.globalData.offlineQueueSize = q.length;
  } catch (e) {}
}

export function dequeueOfflineActions(maxCount) {
  const q = getOfflineQueue();
  if (!q.length) return [];
  const take = typeof maxCount === "number" ? q.slice(0, maxCount) : q.slice();
  const rest = typeof maxCount === "number" ? q.slice(maxCount) : [];
  setOfflineQueue(rest);
  return take;
}

