import { useState, useCallback } from "react";
import { LogEntry, LogLevel, MAX_LOG_ENTRIES } from "../types/logs";

export function useAgentLogs() {
  const [logs, setLogs] = useState<LogEntry[]>([]);

  const addLog = useCallback(
    (level: LogLevel, message: string, detail?: unknown) => {
      if (detail) console.error(detail);
      setLogs((prev) => {
        const next: LogEntry = {
          id: crypto.randomUUID(),
          timestamp: new Date().toISOString(),
          level,
          message,
          detail,
        };
        return [...prev.slice(-(MAX_LOG_ENTRIES - 1)), next];
      });
    },
    [],
  );

  const clearLogs = useCallback(() => setLogs([]), []);

  return { logs, addLog, clearLogs };
}
