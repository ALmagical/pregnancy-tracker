import { useRemoteApi } from "../config";
import { STORAGE_KEYS } from "../utils/constants";
import { Storage } from "../utils/storage";
import { formatDateYYYYMMDD, addDays, nowIso } from "../utils/date";

function ensure(key, initialValue) {
  const v = Storage.get(key, null);
  if (v === null) {
    Storage.set(key, initialValue);
    return initialValue;
  }
  return v;
}

export function initMockDbIfNeeded() {
  if (useRemoteApi()) {
    ensure(STORAGE_KEYS.fmSessions, []);
    ensure(STORAGE_KEYS.fmActiveSession, null);
    ensure(STORAGE_KEYS.contractions, []);
    ensure(STORAGE_KEYS.offlineQueue, []);
    ensure(STORAGE_KEYS.settings, { ai: { contextEnabled: true } });
    return;
  }

  const today = new Date();
  const lastPeriodDate = formatDateYYYYMMDD(addDays(today, -24 * 7 - 3)); // 大约24周+3天
  const userInfo = ensure(STORAGE_KEYS.userInfo, {
    id: "u_local",
    status: "pregnant",
    lastPeriodDate,
    dueDate: formatDateYYYYMMDD(addDays(parseDate(lastPeriodDate), 280)),
    currentWeek: 24,
    currentDay: 3,
    prePregnancyWeight: 55.0,
    currentWeight: 62.0,
    createdAt: nowIso(),
    updatedAt: nowIso()
  });

  ensure(STORAGE_KEYS.checkups, [
    {
      id: "c_001",
      checkupDate: formatDateYYYYMMDD(addDays(today, 3)),
      checkupType: "大排畸",
      hospital: "本地示例医院",
      status: "upcoming",
      hasReport: false,
      reportCount: 0,
      note: ""
    }
  ]);

  ensure(STORAGE_KEYS.weights, [
    {
      id: "w_001",
      weight: userInfo.currentWeight,
      recordedAt: nowIso(),
      week: userInfo.currentWeek,
      day: userInfo.currentDay,
      syncStatus: "synced"
    }
  ]);

  ensure(STORAGE_KEYS.fmSessions, []);
  ensure(STORAGE_KEYS.fmActiveSession, null);
  ensure(STORAGE_KEYS.contractions, []);
  ensure(STORAGE_KEYS.checklist, null);
  ensure(STORAGE_KEYS.settings, { ai: { contextEnabled: true } });
  ensure(STORAGE_KEYS.offlineQueue, []);
}

function parseDate(yyyyMMdd) {
  const [y, m, d] = yyyyMMdd.split("-").map((x) => Number(x));
  return new Date(y, m - 1, d);
}

