/// <reference types="vitest" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// ============================================================
// KALENGO Web — konfigurasi Vite (§10.1)
//
// Anggaran performa §1.4 yang dijaga di sini:
//   - initial JS <= 180 KB gzip  → manualChunks memisahkan vendor
//   - animasi <= 150 KB di beranda → animasi masuk chunk terpisah
//   - CSS/JS diberi hash agar bisa di-cache immutable (§15)
// ============================================================
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // Proxy ke API agar cookie autentikasi same-origin pada pengembangan.
    proxy: {
      "/api": {
        target: process.env.VITE_PROXY_TARGET ?? "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  preview: {
    port: 4173,
  },
  build: {
    target: "es2022",
    sourcemap: true,
    cssCodeSplit: true,
    assetsInlineLimit: 2048,
    reportCompressedSize: true,
    rollupOptions: {
      output: {
        // Pisahkan vendor agar perubahan kode aplikasi tidak membatalkan cache
        // pustaka yang jarang berubah (§15).
        manualChunks(id) {
          if (!id.includes("node_modules")) return undefined;
          if (id.includes("react-dom") || id.includes("/react/") || id.includes("scheduler")) {
            return "vendor-react";
          }
          return "vendor";
        },
      },
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
    coverage: {
      provider: "v8",
      reporter: ["text", "html", "lcov"],
      // Target §1.4: cakupan frontend >= 70% (naik pada M6).
      thresholds: {
        lines: 60,
        functions: 60,
        branches: 60,
        statements: 60,
      },
    },
  },
});
