/**
 * Klien HTTP KALENGO.
 *
 * Kontrak error mengikuti §8.4: setiap respons gagal berbentuk
 *   { "error": { "code": "...", "message": "...", "fields"?: {...}, "request_id"?: "..." } }
 *
 * Pada M0 klien ini sengaja kecil: belum ada retry, cache, atau refresh token.
 * Itu bagian dari M6 saat NotificationProvider dipasang.
 */

export interface ApiErrorField {
  readonly [field: string]: string;
}

export interface ApiErrorPayload {
  readonly code: string;
  readonly message: string;
  readonly fields?: ApiErrorField;
  readonly request_id?: string;
}

export interface ApiErrorEnvelope {
  readonly error: ApiErrorPayload;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields: ApiErrorField | undefined;
  readonly requestId: string | undefined;

  constructor(status: number, payload: ApiErrorPayload) {
    super(payload.message);
    this.name = "ApiError";
    this.status = status;
    this.code = payload.code;
    this.fields = payload.fields;
    this.requestId = payload.request_id;
  }

  /** Kode jaringan/transport: server tidak pernah menjawab. */
  get isNetworkError(): boolean {
    return this.status === 0;
  }

  get isServerError(): boolean {
    return this.status >= 500;
  }
}

const DEFAULT_TIMEOUT_MS = 10_000;

function baseUrl(): string {
  // Kosong = same-origin, mengandalkan proxy Vite pada pengembangan.
  return import.meta.env.VITE_API_BASE_URL ?? "";
}

function isErrorEnvelope(value: unknown): value is ApiErrorEnvelope {
  if (typeof value !== "object" || value === null) return false;
  const candidate = (value as { error?: unknown }).error;
  if (typeof candidate !== "object" || candidate === null) return false;
  const payload = candidate as { code?: unknown; message?: unknown };
  return typeof payload.code === "string" && typeof payload.message === "string";
}

export interface RequestOptions {
  readonly method?: "GET" | "POST" | "PATCH" | "PUT" | "DELETE" | undefined;
  readonly body?: unknown;
  readonly signal?: AbortSignal | undefined;
  readonly timeoutMs?: number | undefined;
  /** Idempotency-Key wajib untuk POST /orders (§12). */
  readonly idempotencyKey?: string | undefined;
  readonly headers?: Record<string, string> | undefined;
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const {
    method = "GET",
    body,
    signal,
    timeoutMs = DEFAULT_TIMEOUT_MS,
    idempotencyKey,
    headers: extraHeaders,
  } = options;

  const controller = new AbortController();

  // Teruskan pembatalan dari pemanggil ke controller internal.
  const onAbort = (): void => controller.abort();
  signal?.addEventListener("abort", onAbort, { once: true });

  const headers = new Headers({ Accept: "application/json" });
  if (body !== undefined) headers.set("Content-Type", "application/json");
  if (idempotencyKey !== undefined) headers.set("Idempotency-Key", idempotencyKey);

  // Ambil token CSRF dari cookie kalengo_csrf (pola double-submit cookie §13.1)
  if (typeof document !== "undefined") {
    const csrfMatch = document.cookie.match(/(?:^|;\s*)kalengo_csrf=([^;]+)/);
    if (csrfMatch?.[1]) {
      headers.set("X-CSRF-Token", decodeURIComponent(csrfMatch[1]));
    }
  }

  if (extraHeaders) {
    for (const [k, v] of Object.entries(extraHeaders)) {
      headers.set(k, v);
    }
  }

  // Dipakai untuk membedakan "habis waktu" dari "dibatalkan pemanggil".
  let timedOut = false;
  const timeout = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);

  try {
    const response = await fetch(`${baseUrl()}${path}`, {
      method,
      headers,
      // Cookie sesi httpOnly dikirim apa adanya (§13.1); token TIDAK disimpan di localStorage.
      credentials: "include",
      signal: controller.signal,
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });

    if (response.status === 204) {
      return undefined as T;
    }

    const raw: unknown = await response.json().catch(() => null);

    if (!response.ok) {
      if (isErrorEnvelope(raw)) {
        throw new ApiError(response.status, raw.error);
      }
      throw new ApiError(response.status, {
        code: "API_SERVER_ERROR",
        message: "Terjadi gangguan di sisi kami. Coba lagi sebentar lagi.",
      });
    }

    return raw as T;
  } catch (cause) {
    if (cause instanceof ApiError) throw cause;
    if (cause instanceof DOMException && cause.name === "AbortError") {
      if (timedOut) {
        // Kode API_TIMEOUT ada di registry backend (§8.6 F) dan pesannya seragam.
        throw new ApiError(0, {
          code: "API_TIMEOUT",
          message: "Koneksi terlalu lama. Periksa jaringan lalu coba lagi.",
        });
      }
      // Dibatalkan pemanggil (mis. komponen dilepas). Ini bukan kegagalan;
      // pemanggil bertanggung jawab mengabaikannya lewat signal.aborted.
      throw new ApiError(0, {
        code: "API_TIMEOUT",
        message: "Permintaan dibatalkan.",
      });
    }
    // NET_OFFLINE adalah kode katalog notifikasi sisi klien (§8.6 F).
    throw new ApiError(0, {
      code: "NET_OFFLINE",
      message: "Kamu offline. Beberapa fitur dimatikan.",
    });
  } finally {
    clearTimeout(timeout);
    signal?.removeEventListener("abort", onAbort);
  }
}
