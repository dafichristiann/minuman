import { useCallback, useEffect, useState } from "react";

import { ApiError } from "./api/client";
import { fetchMeta, fetchReady, type MetaResponse, type ReadyResponse } from "./api/system";
import { devOverlayEnabled } from "./config/env";
import "./styles/diagnostic.css";

/**
 * Halaman diagnostik M0.
 *
 * PENTING: ini BUKAN desain KALENGO. PRD menetapkan urutan kerja — moodboard
 * Pinterest (§5) lebih dulu, baru desain (§6). Halaman ini hanya alat verifikasi
 * bahwa rantai Web → API → PostgreSQL/Redis sudah tersambung. Ia akan digantikan
 * sepenuhnya pada M6.
 *
 * Karena itu tampilannya sengaja minimal, memakai font sistem, tanpa aset merek.
 */

type LoadState = "memuat" | "siap" | "gagal";

interface Diagnostics {
  readonly meta: MetaResponse | null;
  readonly ready: ReadyResponse | null;
  readonly error: ApiError | null;
}

const EMPTY: Diagnostics = { meta: null, ready: null, error: null };

function describeError(error: ApiError): string {
  const parts = [error.message];
  if (error.code) parts.push(`(${error.code})`);
  if (error.requestId) parts.push(`· request_id: ${error.requestId}`);
  return parts.join(" ");
}

export default function App() {
  const [state, setState] = useState<LoadState>("memuat");
  const [data, setData] = useState<Diagnostics>(EMPTY);

  const load = useCallback(async (signal: AbortSignal) => {
    setState("memuat");
    try {
      // Kedua probe dijalankan paralel: halaman ini alat diagnosa, bukan UX.
      const [meta, ready] = await Promise.all([fetchMeta(signal), fetchReady(signal)]);
      setData({ meta, ready, error: null });
      setState("siap");
    } catch (cause) {
      if (signal.aborted) return;
      const error =
        cause instanceof ApiError
          ? cause
          : new ApiError(0, {
              code: "API_SERVER_ERROR",
              message: "Kesalahan tak terduga saat menghubungi API.",
            });
      setData({ meta: null, ready: null, error });
      setState("gagal");
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void load(controller.signal);
    return () => {
      controller.abort();
    };
  }, [load]);

  const refresh = useCallback(() => {
    const controller = new AbortController();
    void load(controller.signal);
  }, [load]);

  return (
    <main className="diag">
      <header className="diag__header">
        <p className="diag__eyebrow">M0 · Fondasi</p>
        <h1 className="diag__title">Status Sistem KALENGO</h1>
        <p className="diag__lead">
          Halaman ini hanya untuk verifikasi teknis. Tampilan sebenarnya dibangun pada M6
          setelah moodboard §5 dan sistem desain §6 disetujui.
        </p>
      </header>

      <section className="diag__panel" aria-labelledby="status-api">
        <div className="diag__panel-head">
          <h2 id="status-api" className="diag__panel-title">
            Koneksi API
          </h2>
          <button type="button" className="diag__button" onClick={refresh} disabled={state === "memuat"}>
            {state === "memuat" ? "Memeriksa…" : "Periksa ulang"}
          </button>
        </div>

        {/* Pembaca layar diberi tahu setiap perubahan status tanpa merebut fokus. */}
        <p role="status" aria-live="polite" className="diag__state">
          {state === "memuat" && "Sedang menghubungi API…"}
          {state === "siap" && "API terjangkau."}
          {state === "gagal" && "API tidak terjangkau."}
        </p>

        {state === "gagal" && data.error !== null && (
          <p className="diag__alert" role="alert">
            {describeError(data.error)}
          </p>
        )}

        {state === "siap" && data.meta !== null && (
          <dl className="diag__facts">
            <div className="diag__fact">
              <dt>Nama aplikasi</dt>
              <dd>{data.meta.app}</dd>
            </div>
            <div className="diag__fact">
              <dt>Lingkungan</dt>
              <dd>{data.meta.env}</dd>
            </div>
            <div className="diag__fact">
              <dt>Versi</dt>
              <dd>
                {data.meta.version} <span className="diag__muted">({data.meta.commit})</span>
              </dd>
            </div>
            <div className="diag__fact">
              <dt>Waktu server (UTC)</dt>
              <dd>{data.meta.server_time}</dd>
            </div>
            <div className="diag__fact">
              <dt>Pembayaran</dt>
              <dd>{data.meta.payment_provider}</dd>
            </div>
            <div className="diag__fact">
              <dt>Pengiriman</dt>
              <dd>{data.meta.shipping_provider}</dd>
            </div>
            <div className="diag__fact">
              <dt>Mode notifikasi</dt>
              <dd>{data.meta.notif_modes.join(", ")}</dd>
            </div>
          </dl>
        )}
      </section>

      {state === "siap" && data.ready !== null && (
        <section className="diag__panel" aria-labelledby="status-dependensi">
          <h2 id="status-dependensi" className="diag__panel-title">
            Dependensi
          </h2>
          <p className="diag__state">
            Status keseluruhan: <strong>{data.ready.status}</strong>
          </p>
          <table className="diag__table">
            <caption className="sr-only">
              Hasil pemeriksaan kesiapan setiap dependensi backend
            </caption>
            <thead>
              <tr>
                <th scope="col">Dependensi</th>
                <th scope="col">Status</th>
                <th scope="col">Latensi</th>
                <th scope="col">Catatan</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(data.ready.checks).map(([name, check]) => (
                <tr key={name}>
                  <th scope="row">{name}</th>
                  <td>{check.status}</td>
                  <td>{check.latency ?? "—"}</td>
                  <td>{check.error ?? "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      {devOverlayEnabled && (
        <footer className="diag__footer">
          <p>
            Overlay diagnostik aktif (<code>VITE_ENABLE_DEV_OVERLAY=true</code>). Nonaktifkan
            sebelum rilis.
          </p>
        </footer>
      )}
    </main>
  );
}
