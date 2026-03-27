function pad2(n) {
  return String(n).padStart(2, "0");
}

export function formatDateYYYYMMDD(date) {
  const d = new Date(date);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
}

export function parseYYYYMMDD(s) {
  if (!s || typeof s !== "string") return null;
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
  if (!m) return null;
  const y = Number(m[1]);
  const mo = Number(m[2]);
  const da = Number(m[3]);
  const d = new Date(y, mo - 1, da);
  if (Number.isNaN(d.getTime())) return null;
  if (d.getFullYear() !== y || d.getMonth() !== mo - 1 || d.getDate() !== da) return null;
  return d;
}

export function addDays(date, days) {
  const d = new Date(date);
  d.setDate(d.getDate() + days);
  return d;
}

export function diffDays(a, b) {
  const da = new Date(a);
  const db = new Date(b);
  const t = Date.UTC(da.getFullYear(), da.getMonth(), da.getDate());
  const u = Date.UTC(db.getFullYear(), db.getMonth(), db.getDate());
  return Math.floor((t - u) / 86400000);
}

export function nowIso() {
  return new Date().toISOString();
}

