/**
 * Konfigurasi runtime frontend.
 *
 * Semua nilai berasal dari import.meta.env (Vite) dan bersifat PUBLIK.
 * Rahasia tidak boleh ada di sini (§13.1).
 */

export type AppEnv = "development" | "staging" | "production";

function readAppEnv(): AppEnv {
  const raw = import.meta.env.VITE_APP_ENV;
  if (raw === "staging" || raw === "production") return raw;
  return "development";
}

export const appEnv: AppEnv = readAppEnv();

export const isProduction = appEnv === "production";

/**
 * Overlay diagnostik M0 (panel status API).
 * Dipaksa mati di produksi apa pun nilai env-nya.
 */
export const devOverlayEnabled = !isProduction && import.meta.env.VITE_ENABLE_DEV_OVERLAY === "true";
