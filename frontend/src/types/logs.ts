export type LogLevel = "error" | "warn" | "info";

export interface LogEntry {
  id: string;
  timestamp: string;
  level: LogLevel;
  message: string;
  detail?: unknown;
}

export const MAX_LOG_ENTRIES = 100;
