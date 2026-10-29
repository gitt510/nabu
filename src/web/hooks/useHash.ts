import { useCallback, useEffect, useState } from "react";

/** URL の #番号（task の id）。開いている task を、reload と共有する link に残す */
export function useHash() {
  const [hash, setHash] = useState(() => decodeURIComponent(location.hash.slice(1)));
  useEffect(() => {
    const on = () => setHash(decodeURIComponent(location.hash.slice(1)));
    window.addEventListener("hashchange", on);
    return () => window.removeEventListener("hashchange", on);
  }, []);
  /** push すると履歴に残り、戻るで前の画面に戻れる（狭い画面の一覧 → 詳細） */
  const set = useCallback((next: string, push = false) => {
    const url = next ? `#${encodeURIComponent(next)}` : location.pathname;
    if (push) history.pushState(null, "", url);
    else history.replaceState(null, "", url);
    setHash(next);
  }, []);
  return [hash, set] as const;
}
