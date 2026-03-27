import { ok, fail } from "../utils/errors";
import { getUserInfo, setUserInfo } from "../utils/storage";
import { validateLastPeriodDate } from "../utils/validation";
import { addDays, diffDays, formatDateYYYYMMDD, nowIso, parseYYYYMMDD } from "../utils/date";

function computePregnancyFromLastPeriod(lastPeriodDate) {
  const d = parseYYYYMMDD(lastPeriodDate);
  if (!d) return null;
  const today = new Date();
  const days = -diffDays(d, today); // today - d
  const week = Math.floor(days / 7);
  const day = days % 7;
  const dueDate = formatDateYYYYMMDD(addDays(d, 280));
  return { currentWeek: week, currentDay: day, dueDate };
}

export async function getUserInfoApi() {
  const userInfo = getUserInfo();
  if (!userInfo) return fail("未设置孕期信息", "E_NOT_FOUND", {});
  return ok(userInfo);
}

export async function updateUserInfoApi(payload) {
  if (payload?.lastPeriodDate) {
    const v = validateLastPeriodDate(payload.lastPeriodDate);
    if (!v.ok) return fail(v.message, v.errorCode, {});
  }

  const current = getUserInfo() || {};
  const next = { ...current, ...payload };

  if (payload?.lastPeriodDate) {
    const computed = computePregnancyFromLastPeriod(payload.lastPeriodDate);
    if (!computed) return fail("末次月经日期无效", "E_PARAM_INVALID", {});
    next.dueDate = computed.dueDate;
    next.currentWeek = computed.currentWeek;
    next.currentDay = computed.currentDay;
  }

  next.updatedAt = nowIso();
  if (!next.createdAt) next.createdAt = nowIso();
  if (!next.id) next.id = "u_local";
  if (!next.status) next.status = "pregnant";

  setUserInfo(next);
  return ok({
    dueDate: next.dueDate,
    currentWeek: next.currentWeek,
    currentDay: next.currentDay
  });
}

