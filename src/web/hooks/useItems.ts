import { useCallback, useEffect, useState } from "react";

import type { Item } from "../../api/schema";
import { fetchItems } from "../api";

/** outline の行全部。write で書き込むと読み直し、断られたら error にその理由を入れる */
export function useItems() {
  const [items, setItems] = useState<Item[] | null>(null);
  const [error, setError] = useState<string>();
  const reload = useCallback(() => {
    fetchItems()
      .then(setItems)
      .catch((e: Error) => setError(e.message));
  }, []);
  useEffect(reload, [reload]);
  /** 書き込みを全部待ってから読み直す。1 つでも断られたら、最初の理由を出す */
  const write = async (...done: Promise<string | undefined>[]) => {
    const problem = (await Promise.all(done)).find(Boolean);
    setError(problem);
    reload();
    return !problem;
  };
  return { items, error, write };
}
