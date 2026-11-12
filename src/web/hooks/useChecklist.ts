import { useCallback, useEffect, useState } from "react";

import type { Item } from "../../api/schema";
import { fetchItems } from "../api";

/** task の checklist。write で書き込むと読み直し、断られたら error にその理由を入れる */
export function useChecklist(taskId: number) {
  const [items, setItems] = useState<Item[] | null>(null);
  const [error, setError] = useState<string>();
  const reload = useCallback(() => {
    fetchItems(taskId)
      .then(setItems)
      .catch((e: Error) => setError(e.message));
  }, [taskId]);
  useEffect(reload, [reload]);
  const write = async (done: Promise<string | undefined>) => {
    const problem = await done;
    setError(problem);
    reload();
    return !problem;
  };
  return { items, error, write };
}
