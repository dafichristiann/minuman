import { apiRequest } from "./client";

/**
 * Bentuk respons endpoint sistem.
 *
 * Endpoint ini SENGAJA di luar kontrak §12 dan TIDAK memakai envelope error:
 * konsumennya orkestrator (healthcheck) dan halaman diagnostik, bukan pengguna.
 * Lihat apps/api/internal/http/handler/system.go.
 */

/** GET /api/v1/meta */
export interface MetaResponse {
  readonly app: string;
  readonly env: string;
  readonly version: string;
  readonly commit: string;
  readonly uptime_seconds: number;
  readonly server_time: string;
  readonly notif_modes: readonly string[];
  readonly payment_provider: string;
  readonly shipping_provider: string;
}

export interface ReadyCheck {
  readonly status: "up" | "down";
  readonly latency?: string;
  readonly error?: string;
}

/** GET /readyz */
export interface ReadyResponse {
  readonly status: "ready" | "degraded";
  readonly version: string;
  readonly uptime_seconds: number;
  readonly checks: Readonly<Record<string, ReadyCheck>>;
}

export function fetchMeta(signal?: AbortSignal): Promise<MetaResponse> {
  return apiRequest<MetaResponse>("/api/v1/meta", signal ? { signal } : {});
}

export function fetchReady(signal?: AbortSignal): Promise<ReadyResponse> {
  return apiRequest<ReadyResponse>("/readyz", signal ? { signal } : {});
}
