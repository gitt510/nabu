import { useCallback, useEffect, useState } from "react";

import type { Habit } from "../../src/fields";
import { fetchHabits } from "../api";

/** day（YYYY-MM-DD）に見た habit 全部。write で書き込むと読み直し、断られたら error にその理由を入れる */
export function useHabits(day: string) {
  const [habits, setHabits] = useState<Habit[] | null>(null);
  const [error, setError] = useState<string>();
  const reload = useCallback(() => {
    fetchHabits(day)
      .then(setHabits)
      .catch((e: Error) => setError(e.message));
  }, [day]);
  useEffect(reload, [reload]);
  const write = async (done: Promise<string | undefined>) => {
    const problem = await done;
    setError(problem);
    reload();
    return !problem;
  };
  return { habits, error, write };
}
