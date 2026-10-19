import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

import App from "./App";

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as unknown as Response;
}

const META = {
  app: "kalengo-api",
  env: "development",
  version: "dev",
  commit: "unknown",
  uptime_seconds: 12,
  server_time: "2026-01-01T00:00:00Z",
  notif_modes: ["normal", "calm", "silent"],
  payment_provider: "midtrans",
  shipping_provider: "biteship",
};

const READY = {
  status: "ready",
  version: "dev",
  uptime_seconds: 12,
  checks: {
    postgres: { status: "up", latency: "1.2ms" },
    redis: { status: "up", latency: "0.4ms" },
  },
};

function routeFetch(): void {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/api/v1/meta")) return jsonResponse(META);
      if (url.includes("/readyz")) return jsonResponse(READY);
      return jsonResponse({ error: { code: "API_NOT_FOUND", message: "tidak ada" } }, 404);
    }),
  );
}

beforeEach(() => {
  routeFetch();
});

describe("App (halaman diagnostik M0)", () => {
  test("menampilkan status memuat lebih dulu", () => {
    // fetch yang tidak pernah selesai agar keadaan "memuat" bisa diperiksa.
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => {})));

    render(<App />);
    expect(screen.getByRole("status")).toHaveTextContent(/menghubungi API/i);
    expect(screen.getByRole("button", { name: /memeriksa/i })).toBeDisabled();
  });

  test("menampilkan data meta setelah API menjawab", async () => {
    render(<App />);

    await waitFor(() => {
      expect(screen.getByRole("status")).toHaveTextContent(/API terjangkau/i);
    });

    expect(screen.getByText("kalengo-api")).toBeInTheDocument();
    expect(screen.getByText("midtrans")).toBeInTheDocument();
    expect(screen.getByText("biteship")).toBeInTheDocument();
    expect(screen.getByText("normal, calm, silent")).toBeInTheDocument();
  });

  test("menampilkan tabel dependensi", async () => {
    render(<App />);

    await waitFor(() => {
      expect(screen.getByRole("table")).toBeInTheDocument();
    });

    expect(screen.getByRole("rowheader", { name: "postgres" })).toBeInTheDocument();
    expect(screen.getByRole("rowheader", { name: "redis" })).toBeInTheDocument();
    expect(screen.getByText("ready")).toBeInTheDocument();
  });

  test("menampilkan alert dengan kode & request_id saat API gagal", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse(
          {
            error: {
              code: "API_SERVER_ERROR",
              message: "Server sedang bermasalah.",
              request_id: "req-9",
            },
          },
          500,
        ),
      ),
    );

    render(<App />);

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Server sedang bermasalah.");
    expect(alert).toHaveTextContent("API_SERVER_ERROR");
    expect(alert).toHaveTextContent("req-9");
  });

  test("punya judul halaman yang jelas", async () => {
    render(<App />);
    expect(
      screen.getByRole("heading", { level: 1, name: /status sistem kalengo/i }),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("status")).toHaveTextContent(/API terjangkau/i);
    });
  });
});
