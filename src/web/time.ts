// scheduled is RFC3339 with an offset, or a date alone (YYYY-MM-DD) for the whole day in Asia/Tokyo
const dateOnly = (s: string) => /^\d{4}-\d{2}-\d{2}$/.test(s);
export const instant = (s: string) => new Date(dateOnly(s) ? `${s}T00:00:00+09:00` : s);

const day = { timeZone: "Asia/Tokyo", month: "numeric", day: "numeric" } as const;
/** 日付だけなら 10/28、時刻があれば 10/28 18:00 */
export const fmt = (iso: string) => {
  const d = instant(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return dateOnly(iso) ? d.toLocaleDateString("ja-JP", day) : d.toLocaleString("ja-JP", { ...day, hour: "2-digit", minute: "2-digit" });
};
export const fmtDay = (iso: string) => fmt(iso).split(" ")[0];

// 暦の 1 日は Asia/Tokyo の YYYY-MM-DD で表す。文字列のまま大小を比べられる
const KEY = new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Tokyo", year: "numeric", month: "2-digit", day: "2-digit" });
export const todayKey = () => KEY.format(new Date());
export const dayOf = (iso: string) => KEY.format(instant(iso));
/** 予定日が今日より前 */
export const isLate = (iso: string) => dayOf(iso) < todayKey();
export const addDays = (key: string, n: number) => {
  const [y, m, d] = key.split("-").map(Number);
  return new Date(Date.UTC(y, m - 1, d + n)).toISOString().slice(0, 10);
};
/** Mon, Oct 26, 2026 */
export const dayLabel = (key: string) =>
  new Date(`${key}T00:00:00Z`).toLocaleDateString("en-US", { timeZone: "UTC", year: "numeric", month: "short", day: "numeric", weekday: "short" });
/** 時刻のある予定は 18:00、日付だけの予定は null */
export const timeOf = (iso: string) => (dateOnly(iso) ? null : instant(iso).toLocaleTimeString("ja-JP", { timeZone: "Asia/Tokyo", hour: "2-digit", minute: "2-digit" }));

/** ticket や PR の URL を、短い名前にする（Wrike 123、repo#45） */
export const ticket = (u: string) => {
  try {
    const url = new URL(u);
    if (url.hostname.includes("wrike")) return `Wrike ${url.searchParams.get("id") ?? ""}`.trim();
    if (url.hostname === "github.com") {
      const [, , repo, , num] = url.pathname.split("/");
      return num ? `${repo}#${num}` : repo;
    }
    return url.hostname;
  } catch {
    return u;
  }
};
