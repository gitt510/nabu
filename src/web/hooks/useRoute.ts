import { useCallback, useEffect, useState } from "react";

export type Page = "home" | "tasks" | "habits";
/** URL が指す画面。id は Tasks で開いている task の番号で、開いていなければ "" */
export type Route = { page: Page; id: string };

const parse = (path: string): Route => {
  const [, page, id = ""] = path.split("/");
  if (page === "tasks") return { page, id: decodeURIComponent(id) };
  if (page === "habits") return { page, id: "" };
  return { page: "home", id: "" };
};

const format = ({ page, id }: Route) =>
  page === "home" ? "/" : id ? `/${page}/${encodeURIComponent(id)}` : `/${page}`;

/**
 * 画面を URL の path に持つ：/ は Home、/tasks は Tasks、/tasks/6 は #6 を開いた Tasks、/habits は Habits。
 * reload しても同じ画面に戻り、画面ごとの link を共有でき、ブラウザの戻るで前の画面に戻れる
 */
export function useRoute() {
  const [route, setRoute] = useState(() => {
    // 以前の /#6 の link は /tasks/6 に読み替える
    const old = decodeURIComponent(location.hash.slice(1));
    if (old) history.replaceState(null, "", format({ page: "tasks", id: old }));
    return parse(location.pathname);
  });
  useEffect(() => {
    const on = () => setRoute(parse(location.pathname));
    window.addEventListener("popstate", on);
    return () => window.removeEventListener("popstate", on);
  }, []);
  /** push すると履歴に残り、戻るで前の画面に戻れる。push しなければ今の履歴を書き換える */
  const go = useCallback((next: Route, push = true) => {
    const url = format(next);
    if (url !== location.pathname) {
      if (push) history.pushState(null, "", url);
      else history.replaceState(null, "", url);
    }
    setRoute(next);
  }, []);
  return [route, go] as const;
}
