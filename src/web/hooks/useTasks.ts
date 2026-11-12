import { useCallback, useEffect, useState } from "react";

import { fetchTasks, type TaskView } from "../api";

/** API の task 全部。書き込んだあとは reload で読み直し、API を見せるものの唯一の出どころにする */
export function useTasks() {
  const [tasks, setTasks] = useState<TaskView[] | null>(null);
  const [error, setError] = useState<string>();
  const reload = useCallback(() => {
    fetchTasks()
      .then((t) => {
        setTasks(t);
        setError(undefined);
      })
      .catch((e: Error) => setError(e.message));
  }, []);
  useEffect(reload, [reload]);
  return { tasks, error, reload };
}
