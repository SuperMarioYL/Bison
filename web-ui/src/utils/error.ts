import { AxiosError } from 'axios';

/**
 * Extract a human-readable message from an API/network error.
 *
 * The backend wraps errors as `{ "error": "..." }`; this centralizes that
 * extraction so callers no longer copy the same `err?.response?.data?.error`
 * dance (previously duplicated across ~13 sites).
 */
export function getApiErrorMessage(err: unknown, fallback = '操作失败'): string {
  if (err instanceof AxiosError) {
    const data = err.response?.data as { error?: string; message?: string } | undefined;
    return data?.error || data?.message || err.message || fallback;
  }
  if (err instanceof Error) {
    return err.message || fallback;
  }
  return fallback;
}
