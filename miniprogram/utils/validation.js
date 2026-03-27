import { parseYYYYMMDD, diffDays } from "./date";

export function validateLastPeriodDate(lastPeriodDate) {
  const d = parseYYYYMMDD(lastPeriodDate);
  if (!d) {
    return { ok: false, errorCode: "E_PARAM_INVALID", message: "请输入有效的末次月经日期" };
  }
  const today = new Date();
  const daysFromToday = diffDays(d, today); // d - today
  if (daysFromToday > 0) {
    return { ok: false, errorCode: "E_PARAM_INVALID", message: "末次月经日期不能晚于今天" };
  }
  if (daysFromToday < -365 * 2) {
    return { ok: false, errorCode: "E_PARAM_INVALID", message: "末次月经日期过早，请检查后重试" };
  }
  return { ok: true };
}

export function validateWeightKg(weight) {
  const w = Number(weight);
  if (!Number.isFinite(w)) {
    return { ok: false, errorCode: "E_PARAM_INVALID", message: "请输入有效体重" };
  }
  const rounded = Math.round(w * 10) / 10;
  if (rounded < 30 || rounded > 200) {
    return {
      ok: false,
      errorCode: "E_WEIGHT_UNREASONABLE",
      message: "体重似乎超出常见范围，是否确认保存？",
      data: { rounded }
    };
  }
  return { ok: true, data: { rounded } };
}

