/**
 * Uji aksesibilitas otomatis dengan axe-core (§14).
 *
 * Dijalankan lewat `npm run test:a11y`. Pada M6 berkas ini diperluas untuk
 * menguji SETIAP komponen notifikasi — persyaratan §8.8: setiap kode katalog
 * punya test render + axe.
 *
 * Catatan: axe-core di jsdom tidak bisa mengukur warna (tidak ada layout),
 * sehingga aturan color-contrast otomatis dilewati. Kontras tetap wajib diuji
 * manual / lewat Lighthouse pada M6.
 */
import { render } from "@testing-library/react";
import axe from "axe-core";
import { expect, test, vi } from "vitest";

import App from "../App";

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
  uptime_seconds: 3,
  server_time: "2026-01-01T00:00:00Z",
  notif_modes: ["normal", "calm", "silent"],
  payment_provider: "midtrans",
  shipping_provider: "biteship",
};

const READY = {
  status: "ready",
  version: "dev",
  uptime_seconds: 3,
  checks: { postgres: { status: "up" }, redis: { status: "up" } },
};

test("halaman diagnostik bebas pelanggaran axe saat data termuat", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) =>
      String(input).includes("/readyz") ? jsonResponse(READY) : jsonResponse(META),
    ),
  );

  const { container, findByRole } = render(<App />);
  await findByRole("table");

  const results = await axe.run(container);
  expect(results.violations).toEqual([]);
});

test("halaman diagnostik bebas pelanggaran axe saat API gagal", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () =>
      jsonResponse({ error: { code: "API_SERVER_ERROR", message: "Server sedang bermasalah." } }, 500),
    ),
  );

  const { container, findByRole } = render(<App />);
  await findByRole("alert");

  const results = await axe.run(container);
  expect(results.violations).toEqual([]);
});
