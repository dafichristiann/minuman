import { describe, expect, test, vi } from "vitest";

import { ApiError, apiRequest } from "./client";

/** Respons palsu; tidak bergantung pada global Response milik jsdom/Node. */
function fakeResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as unknown as Response;
}

describe("apiRequest", () => {
  test("mengembalikan body JSON saat berhasil", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => fakeResponse({ app: "kalengo-api" })));

    const result = await apiRequest<{ app: string }>("/api/v1/meta");

    expect(result.app).toBe("kalengo-api");
  });

  test("mengirim cookie sesi (credentials: include)", async () => {
    const fetchMock = vi.fn(async () => fakeResponse({}));
    vi.stubGlobal("fetch", fetchMock);

    await apiRequest("/api/v1/me");

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    expect(init?.credentials).toBe("include");
  });

  test("meneruskan Idempotency-Key untuk POST /orders", async () => {
    const fetchMock = vi.fn(async () => fakeResponse({}, 201));
    vi.stubGlobal("fetch", fetchMock);

    await apiRequest("/api/v1/orders", {
      method: "POST",
      body: { items: [] },
      idempotencyKey: "abc-123",
    });

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    const headers = init?.headers as Headers;
    expect(headers.get("Idempotency-Key")).toBe("abc-123");
    expect(headers.get("Content-Type")).toBe("application/json");
  });

  test("mengembalikan undefined untuk 204", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => fakeResponse(null, 204)));

    await expect(apiRequest("/api/v1/cart/items/x", { method: "DELETE" })).resolves.toBeUndefined();
  });

  test("melempar ApiError dengan kode dari envelope backend", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        fakeResponse(
          {
            error: {
              code: "CART_PRICE_CHANGED",
              message: "Harga Kopi Susu berubah dari Rp12.000 menjadi Rp14.000.",
              fields: { variant_id: "harga berubah" },
              request_id: "req-1",
            },
          },
          409,
        ),
      ),
    );

    const error = await apiRequest("/api/v1/orders", { method: "POST" }).catch((e: unknown) => e);

    expect(error).toBeInstanceOf(ApiError);
    const apiError = error as ApiError;
    expect(apiError.status).toBe(409);
    expect(apiError.code).toBe("CART_PRICE_CHANGED");
    expect(apiError.requestId).toBe("req-1");
    expect(apiError.fields?.["variant_id"]).toBe("harga berubah");
  });

  test("memakai kode umum bila envelope error tidak sesuai kontrak", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => fakeResponse({ pesan: "rusak" }, 502)));

    const error = (await apiRequest("/api/v1/products").catch((e: unknown) => e)) as ApiError;

    expect(error).toBeInstanceOf(ApiError);
    expect(error.code).toBe("API_SERVER_ERROR");
    expect(error.isServerError).toBe(true);
  });

  test("mengubah kegagalan jaringan menjadi NET_OFFLINE", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("Failed to fetch");
      }),
    );

    const error = (await apiRequest("/api/v1/products").catch((e: unknown) => e)) as ApiError;

    expect(error.code).toBe("NET_OFFLINE");
    expect(error.status).toBe(0);
    expect(error.isNetworkError).toBe(true);
  });

  test("mengubah abort karena timeout menjadi API_TIMEOUT", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        (_url: string, init?: RequestInit) =>
          new Promise<Response>((_resolve, reject) => {
            init?.signal?.addEventListener("abort", () => {
              reject(new DOMException("aborted", "AbortError"));
            });
          }),
      ),
    );

    const error = (await apiRequest("/api/v1/products", { timeoutMs: 10 }).catch(
      (e: unknown) => e,
    )) as ApiError;

    expect(error.code).toBe("API_TIMEOUT");
  });

  test("meneruskan pembatalan dari pemanggil ke fetch", async () => {
    const controller = new AbortController();
    let sawAbort = false;
    vi.stubGlobal(
      "fetch",
      vi.fn(
        (_url: string, init?: RequestInit) =>
          new Promise<Response>((_resolve, reject) => {
            init?.signal?.addEventListener("abort", () => {
              sawAbort = true;
              reject(new DOMException("aborted", "AbortError"));
            });
          }),
      ),
    );

    const promise = apiRequest("/api/v1/products", { signal: controller.signal }).catch(
      (e: unknown) => e as ApiError,
    );
    controller.abort();
    await promise;

    expect(sawAbort).toBe(true);
  });
});
