// Validasi minimal kontrak OpenAPI: YAML harus bisa di-parse dan setiap
// $ref internal harus menunjuk komponen yang benar-benar ada.
//
// Dipakai sebagai gerbang cepat sebelum klien dibangkitkan (§10.2). Bukan
// pengganti validator penuh; hanya menangkap kesalahan struktural yang umum.
import { readFileSync } from "node:fs";
import { load } from "js-yaml";

const path = new URL("../../api/openapi.yaml", import.meta.url);
const doc = load(readFileSync(path, "utf8"));

const dangling = [];
const missing = [];

function walk(node) {
  if (Array.isArray(node)) {
    node.forEach(walk);
    return;
  }
  if (node && typeof node === "object") {
    for (const [key, value] of Object.entries(node)) {
      if (key === "$ref" && typeof value === "string") {
        const match = /^#\/components\/([^/]+)\/([^/]+)$/.exec(value);
        if (!match) continue;
        const [, group, name] = match;
        if (!doc.components?.[group]?.[name]) dangling.push(value);
      }
      walk(value);
    }
  }
}

walk(doc);

const requiredPaths = [
  "/api/v1/auth/verify-email/resend",
  "/api/v1/me/addresses/{id}/default",
];
for (const p of requiredPaths) {
  if (!doc.paths?.[p]) missing.push(p);
}
if (!doc.paths?.["/api/v1/wishlist/{productId}"]?.get) {
  missing.push("GET /api/v1/wishlist/{productId}");
}

const m1 = Object.values(doc.paths ?? {}).flatMap((item) =>
  Object.values(item).filter((op) => op && typeof op === "object" && op["x-milestone"] === "M1"),
);

const result = {
  paths: Object.keys(doc.paths ?? {}).length,
  schemas: Object.keys(doc.components?.schemas ?? {}).length,
  m1Operations: m1.length,
  danglingRefs: dangling,
  missingPaths: missing,
};

process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);

if (dangling.length > 0 || missing.length > 0) {
  process.exitCode = 1;
}
