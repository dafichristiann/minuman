#!/usr/bin/env node
/**
 * Penjaga anggaran performa (§1.4).
 *
 * Diukur pada hasil `vite build` (dist/), bukan pada dev server:
 *   - JS awal (entry + preload)   <= 180 KB gzip
 *   - CSS total                   <=  40 KB gzip
 *   - Setiap file JS individual   <= 150 KB gzip
 *   - Total aset JS               <= 400 KB gzip  (jaring pengaman)
 *
 * Skrip ini GAGAL (exit 1) bila salah satu anggaran terlampaui. CI memanggilnya
 * setelah build; jangan dilewati dengan alasan "sementara".
 */
import { readFile, readdir, stat } from "node:fs/promises";
import { extname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { gzipSync } from "node:zlib";

const DIST = fileURLToPath(new URL("../dist/", import.meta.url));

const KB = 1024;
const BUDGET = {
  initialJs: 180 * KB,
  initialCss: 40 * KB,
  singleJs: 150 * KB,
  totalJs: 400 * KB,
};

/** @param {number} bytes */
const kb = (bytes) => `${(bytes / KB).toFixed(1)} KB gzip`;

/**
 * @param {string} dir
 * @returns {Promise<string[]>}
 */
async function walk(dir) {
  const entries = await readdir(dir, { withFileTypes: true });
  /** @type {string[]} */
  const files = [];
  for (const entry of entries) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) files.push(...(await walk(full)));
    else files.push(full);
  }
  return files;
}

/** @param {string} file */
async function gzipSize(file) {
  return gzipSync(await readFile(file), { level: 9 }).length;
}

/** @param {string} file */
async function exists(file) {
  try {
    await stat(file);
    return true;
  } catch {
    return false;
  }
}

let files;
try {
  files = await walk(DIST);
} catch {
  console.error("x dist/ belum ada. Jalankan `npm run build` lebih dulu.");
  process.exit(1);
}

const jsFiles = files.filter((f) => extname(f) === ".js");
const cssFiles = files.filter((f) => extname(f) === ".css");

if (jsFiles.length === 0) {
  console.error("x Tidak ada berkas JS di dist/. Build gagal?");
  process.exit(1);
}

const problems = [];
const notes = [];

// ---- Setiap file JS individual + total ----
let totalJs = 0;
let largestJs = 0;
for (const file of jsFiles) {
  const size = await gzipSize(file);
  totalJs += size;
  if (size > largestJs) largestJs = size;
  if (size > BUDGET.singleJs) {
    problems.push(`${relative(process.cwd(), file)} = ${kb(size)} > batas ${kb(BUDGET.singleJs)}`);
  }
}

if (totalJs > BUDGET.totalJs) {
  problems.push(`Total JS = ${kb(totalJs)} > batas ${kb(BUDGET.totalJs)}`);
}

// ---- JS & CSS awal menurut index.html ----
let initialJs = 0;
let initialCss = 0;
const indexPath = join(DIST, "index.html");

if (await exists(indexPath)) {
  const html = await readFile(indexPath, "utf8");
  const referenced = [...html.matchAll(/(?:src|href)="([^"]+\.(?:js|css))"/g)]
    .map((match) => match[1])
    .filter((value) => typeof value === "string" && value.length > 0);

  for (const ref of referenced) {
    const full = join(DIST, ref.replace(/^\//, ""));
    if (!(await exists(full))) continue;
    const size = await gzipSize(full);
    if (extname(full) === ".js") initialJs += size;
    else initialCss += size;
  }
} else {
  notes.push("index.html tidak ditemukan; anggaran JS awal dilewati.");
}

if (initialJs > BUDGET.initialJs) {
  problems.push(`JS awal = ${kb(initialJs)} > batas ${kb(BUDGET.initialJs)}`);
}

let totalCss = 0;
for (const file of cssFiles) totalCss += await gzipSize(file);
if (totalCss > BUDGET.initialCss) {
  problems.push(`CSS total = ${kb(totalCss)} > batas ${kb(BUDGET.initialCss)}`);
}

// ---- Laporan ----
console.log("Anggaran performa (§1.4)");
console.log(`  JS awal      : ${kb(initialJs).padStart(12)} / ${kb(BUDGET.initialJs)}`);
console.log(`  CSS total    : ${kb(totalCss).padStart(12)} / ${kb(BUDGET.initialCss)}`);
console.log(`  Total JS     : ${kb(totalJs).padStart(12)} / ${kb(BUDGET.totalJs)}`);
console.log(`  JS terbesar  : ${kb(largestJs).padStart(12)} / ${kb(BUDGET.singleJs)}`);
for (const note of notes) console.log(`  catatan: ${note}`);

if (problems.length > 0) {
  console.error("\nx Anggaran terlampaui:");
  for (const problem of problems) console.error(`  - ${problem}`);
  console.error("\nPerbaiki dengan code-splitting, lazy-load, atau kurangi dependensi.");
  process.exit(1);
}

console.log("\nOK: semua anggaran performa terpenuhi.");
