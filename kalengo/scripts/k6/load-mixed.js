// ============================================================
// KALENGO — uji beban campuran (§13.3, anggaran §1.4)
//
//   k6 run scripts/k6/load-mixed.js
//   BASE_URL=http://localhost:8080 k6 run scripts/k6/load-mixed.js
//
// Ambang batas (§1.4):
//   baca  p95 <= 300 ms
//   tulis p95 <= 600 ms
//   throughput 200 rps
//   error rate < 1%
//
// CATATAN M0: sebagian endpoint di bawah masih menjawab 501 (belum
// diimplementasikan sampai M2). Skrip ini tetap dijalankan sebagai bukti
// infrastruktur (nginx → api → postgres/redis) sanggup menerima beban;
// anggaran p95 baru bermakna setelah endpoint domain ada (M2).
//
// Skrip ini SENGAJA tidak melakukan mutasi yang mengubah data bisnis
// (tidak membuat pesanan sungguhan). Uji tulis memakai endpoint aman:
// POST /api/v1/cart/items akan ditambahkan pada M1 dengan tag `write`.
// ============================================================

import http from "k6/http";
import { check, group, sleep } from "k6";
import { Trend, Rate, Counter } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

// ---------- metrik kustom ----------
const readDuration = new Trend("kalengo_read_duration", true);
const writeDuration = new Trend("kalengo_write_duration", true);
const serverErrorRate = new Rate("kalengo_server_error_rate");
const notImplemented = new Counter("kalengo_not_implemented");

// ---------- opsi ----------
export const options = {
  scenarios: {
    // Beban baca: meniru pengunjung menelusuri katalog (§1.4 p95 <= 300 ms).
    browse: {
      executor: "ramping-arrival-rate",
      exec: "browse",
      startRate: 20,
      timeUnit: "1s",
      preAllocatedVUs: 50,
      maxVUs: 200,
      stages: [
        { target: 100, duration: "30s" },
        { target: 200, duration: "1m" },
        { target: 200, duration: "2m" },
        { target: 0, duration: "30s" },
      ],
    },
    // Beban tulis ringan (§1.4 p95 <= 600 ms). Aktif penuh setelah M2.
    write: {
      executor: "constant-arrival-rate",
      exec: "write",
      rate: 10,
      timeUnit: "1s",
      duration: "4m",
      preAllocatedVUs: 20,
      maxVUs: 60,
    },
  },
  thresholds: {
    // Gerbang keras: bila gagal, k6 keluar dengan kode != 0.
    "http_req_failed": ["rate<0.01"],
    "kalengo_server_error_rate": ["rate<0.01"],

    // Anggaran §1.4
    "kalengo_read_duration{kind:read}": ["p(95)<300"],
    "kalengo_write_duration{kind:write}": ["p(95)<600"],

    // Jalur kritis tidak boleh lambat
    "http_req_duration{endpoint:meta}": ["p(95)<100"],
    "http_req_duration{endpoint:healthz}": ["p(95)<50"],
  },
  summaryTrendStats: ["avg", "min", "med", "p(90)", "p(95)", "p(99)", "max"],
};

// ---------- helper ----------

function record(response, kind, endpoint) {
  const tags = { kind, endpoint };

  if (kind === "write") writeDuration.add(response.timings.duration, tags);
  else readDuration.add(response.timings.duration, tags);

  if (response.status === 501) {
    // Belum diimplementasikan (M0/M1). Bukan kesalahan sistem.
    notImplemented.add(1, { endpoint });
    return;
  }

  if (response.status >= 500) {
    serverErrorRate.add(1, { endpoint });
    console.error(`${endpoint} -> ${response.status}: ${String(response.body).slice(0, 200)}`);
  } else {
    serverErrorRate.add(0, { endpoint });
  }
}

function get(path, endpoint) {
  const response = http.get(`${BASE_URL}${path}`, {
    tags: { endpoint },
    headers: { Accept: "application/json" },
  });
  record(response, "read", endpoint);
  return response;
}

// ---------- skenario: baca ----------

export function browse() {
  group("beranda & katalog", () => {
    const meta = get("/api/v1/meta", "meta");
    check(meta, { "meta 200": (r) => r.status === 200 });

    const list = get("/api/v1/products?limit=24", "products_list");
    check(list, {
      "products 200 atau 501": (r) => r.status === 200 || r.status === 501,
    });

    const cats = get("/api/v1/categories", "categories_list");
    check(cats, { "categories 200 atau 501": (r) => r.status === 200 || r.status === 501 });
  });

  group("detail produk", () => {
    const detail = get("/api/v1/products/kalengo-soda-lemon-330", "product_detail");
    check(detail, { "detail 200/404/501": (r) => [200, 404, 501].includes(r.status) });
  });

  group("pencarian", () => {
    const search = get("/api/v1/search?q=kopi", "search");
    check(search, { "search 200 atau 501": (r) => r.status === 200 || r.status === 501 });
  });

  // Jeda berpikir seperti manusia (0,5–1,5 dtk) agar pola trafik realistis.
  sleep(0.5 + Math.random());
}

// ---------- skenario: tulis ----------

export function write() {
  // Uji tulis memakai endpoint yang tidak mengubah data bisnis nyata.
  // Setelah M1, ganti dengan POST /api/v1/cart/items beserta cookie sesi.
  const response = http.post(
    `${BASE_URL}/api/v1/orders`,
    JSON.stringify({ items: [] }),
    {
      tags: { endpoint: "orders_create" },
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        // Idempotency-Key wajib untuk POST /orders (§12).
        "Idempotency-Key": `${__VU}-${__ITER}-${Date.now()}`,
      },
    },
  );

  record(response, "write", "orders_create");
  check(response, {
    "orders ditolak dengan kode yang jelas (bukan 5xx)": (r) => r.status < 500,
  });

  sleep(1);
}

// ---------- ringkasan ----------

export function handleSummary(data) {
  const readP95 = data.metrics["kalengo_read_duration"]?.values?.["p(95)"] ?? 0;
  const writeP95 = data.metrics["kalengo_write_duration"]?.values?.["p(95)"] ?? 0;
  const notImpl = data.metrics["kalengo_not_implemented"]?.values?.count ?? 0;

  const lines = [
    "",
    "Ringkasan KALENGO (§1.4)",
    `  baca  p95 : ${readP95.toFixed(1).padStart(8)} ms  (anggaran <= 300)`,
    `  tulis p95 : ${writeP95.toFixed(1).padStart(8)} ms  (anggaran <= 600)`,
    `  endpoint 501 (belum ada): ${notImpl}`,
    "",
  ].join("\n");

  console.log(lines);

  return {
    stdout: JSON.stringify(data, null, 2),
  };
}
