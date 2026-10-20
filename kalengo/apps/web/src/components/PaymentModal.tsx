import React, { useState } from "react";
import type { Order } from "../api/checkout";
import { useToast } from "../context/useToast";
import { formatIdr } from "../utils/format";
import { CelebrationConfetti } from "./CelebrationConfetti";
import { CanSodaLemon } from "./mascots/MascotCans";

export interface PaymentModalProps {
  order: Order | null;
  isOpen: boolean;
  onClose: () => void;
  onPaymentSuccess?: (order: Order) => void;
}

export const PaymentModal: React.FC<PaymentModalProps> = ({
  order,
  isOpen,
  onClose,
  onPaymentSuccess,
}) => {
  const [selectedMethod, setSelectedMethod] = useState<string>("qris");
  const [isPaid, setIsPaid] = useState<boolean>(order?.status === "paid");
  const [isProcessing, setIsProcessing] = useState<boolean>(false);
  const toast = useToast();

  if (!isOpen || !order) return null;

  const handleSimulatePayment = async () => {
    setIsProcessing(true);

    // Kirim webhook settlement ke backend jika ini pesanan server nyata
    if (order.order_no.startsWith("KLG-") && typeof crypto !== "undefined" && crypto.subtle) {
      try {
        const statusCode = "200";
        const grossAmount = `${order.total}.00`;
        const serverKey = "dummy-server-key";
        const enc = new TextEncoder();
        const raw = `${order.order_no}${statusCode}${grossAmount}${serverKey}`;
        const buf = await crypto.subtle.digest("SHA-512", enc.encode(raw));
        const signature = Array.from(new Uint8Array(buf))
          .map((b) => b.toString(16).padStart(2, "0"))
          .join("");

        await fetch("/api/v1/webhooks/midtrans", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            order_id: order.order_no,
            status_code: statusCode,
            gross_amount: grossAmount,
            transaction_status: "settlement",
            signature_key: signature,
          }),
        });
      } catch {
        // Lanjutkan konfirmasi UI
      }
    }

    setTimeout(() => {
      setIsProcessing(false);
      setIsPaid(true);
      toast.success(
        "Pembayaran berhasil dikonfirmasi! Pesanan sedang dikemas dingin 🥤",
        "Lunas",
      );
      const updatedOrder: Order = {
        ...order,
        status: "paid",
        paid_at: new Date().toISOString(),
      };
      onPaymentSuccess?.(updatedOrder);
    }, 900);
  };

  const paymentMethods = [
    { id: "qris", name: "QRIS Instant", logo: "📱", desc: "BCA, GoPay, OVO, ShopeePay, Dana" },
    { id: "bca_va", name: "BCA Virtual Account", logo: "🏦", desc: "No. VA: 8271 0812 9923 11" },
    { id: "mandiri_va", name: "Mandiri Bill", logo: "💳", desc: "Kode Perusahaan: 70012" },
    { id: "gopay", name: "GoPay / GoPay Later", logo: "🟢", desc: "Langsung buka aplikasi Gojek" },
  ];

  return (
    <div
      className="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="payment-modal-title"
    >
      <div className="modal-window" style={{ maxWidth: "540px", position: "relative", overflow: "hidden" }}>
        {isPaid ? (
          // Layar Bukti Pembayaran Berhasil (Success Receipt)
          <div style={{ position: "relative" }}>
            <CelebrationConfetti active={isPaid} />

            <header className="modal-header" style={{ backgroundColor: "var(--mint, #3ddc97)" }}>
              <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                <span style={{ fontSize: "1.5rem" }} aria-hidden="true">🎉</span>
                <h2 id="payment-modal-title" className="modal-title">
                  Pembayaran Berhasil!
                </h2>
              </div>
            </header>

            <div className="modal-body" style={{ textAlign: "center", display: "flex", flexDirection: "column", alignItems: "center", gap: "1rem" }}>
              <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: "1rem", margin: "0.5rem 0" }}>
                <div className="bouncy-checkmark-circle" aria-hidden="true">
                  ✓
                </div>
                <div className="mascot-anim--cheer">
                  <CanSodaLemon size={140} alt="Maskot Kalengo Merayakan Sukses" />
                </div>
              </div>

              <h3 style={{ fontFamily: "var(--font-display)", fontSize: "1.375rem", margin: 0 }}>
                Hore! Minumanmu Segera Dikirim 🚚
              </h3>
              <p style={{ color: "#4a4a63", fontSize: "0.875rem", margin: 0, maxWidth: "420px" }}>
                Pesanan nomor <strong>{order.order_no}</strong> sudah kami terima dan sedang dikemas dingin di gudang KALENGO.
              </p>

              <div
                style={{
                  width: "100%",
                  backgroundColor: "var(--cream, #fff4dc)",
                  border: "2.5px solid var(--ink, #1b1b2f)",
                  borderRadius: "var(--radius-md, 0.875rem)",
                  padding: "1rem",
                  textAlign: "left",
                  fontSize: "0.875rem",
                }}
              >
                <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "0.35rem" }}>
                  <span style={{ color: "#666" }}>Nomor Pesanan</span>
                  <span style={{ fontWeight: 800 }}>{order.order_no}</span>
                </div>
                <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "0.35rem" }}>
                  <span style={{ color: "#666" }}>Total Dibayar</span>
                  <span style={{ fontWeight: 800 }}>{formatIdr(order.total)}</span>
                </div>
                <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "0.35rem" }}>
                  <span style={{ color: "#666" }}>Metode</span>
                  <span style={{ fontWeight: 800, textTransform: "uppercase" }}>{selectedMethod}</span>
                </div>
                <div style={{ display: "flex", justifyContent: "space-between" }}>
                  <span style={{ color: "#666" }}>Status</span>
                  <span className="badge badge--mint">LUNAS</span>
                </div>
              </div>
            </div>

            <footer className="modal-footer" style={{ justifyContent: "center" }}>
              <button
                type="button"
                className="btn btn--primary btn--pill"
                onClick={onClose}
              >
                Selesai & Belanja Lagi 🥤
              </button>
            </footer>
          </div>
        ) : (
          // Layar Pilihan Pembayaran (Midtrans Snap Simulation)
          <div>
            <header className="modal-header" style={{ backgroundColor: "var(--lemon, #ffd23f)" }}>
              <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                <span style={{ fontSize: "1.5rem" }} aria-hidden="true">💳</span>
                <div>
                  <h2 id="payment-modal-title" className="modal-title" style={{ fontSize: "1.25rem" }}>
                    Bayar Pesanan
                  </h2>
                  <span style={{ fontSize: "0.75rem", fontWeight: 700, color: "#4a4a63" }}>
                    No: {order.order_no}
                  </span>
                </div>
              </div>
              <button
                type="button"
                className="btn btn--sm btn--secondary"
                onClick={onClose}
                aria-label="Tutup pembayaran"
                style={{ padding: "0.35rem 0.65rem" }}
              >
                ✕
              </button>
            </header>

            <div className="modal-body" style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  padding: "0.875rem",
                  backgroundColor: "var(--cream, #fff4dc)",
                  border: "2px solid #1b1b2f",
                  borderRadius: "0.5rem",
                }}
              >
                <div>
                  <div style={{ fontSize: "0.75rem", color: "#6a6a80", fontWeight: 700 }}>Total Tagihan</div>
                  <div style={{ fontSize: "1.5rem", fontWeight: 900, color: "var(--ink, #1b1b2f)" }}>
                    {formatIdr(order.total)}
                  </div>
                </div>
                <span className="badge badge--hot">Sisa Waktu 14:59</span>
              </div>

              <div>
                <label style={{ display: "block", fontSize: "0.8125rem", fontWeight: 800, marginBottom: "0.5rem" }}>
                  Pilih Kanal Pembayaran (Midtrans Snap):
                </label>
                <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
                  {paymentMethods.map((pm) => {
                    const isSelected = selectedMethod === pm.id;
                    return (
                      <button
                        key={pm.id}
                        type="button"
                        className={`btn ${isSelected ? "btn--lemon" : "btn--secondary"}`}
                        style={{
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "flex-start",
                          gap: "0.75rem",
                          padding: "0.75rem 1rem",
                          textAlign: "left",
                        }}
                        onClick={() => setSelectedMethod(pm.id)}
                      >
                        <span style={{ fontSize: "1.375rem" }} aria-hidden="true">{pm.logo}</span>
                        <div style={{ flex: 1 }}>
                          <div style={{ fontWeight: 900, fontSize: "0.9375rem" }}>{pm.name}</div>
                          <div style={{ fontSize: "0.75rem", color: "#666", fontWeight: 600 }}>{pm.desc}</div>
                        </div>
                        {isSelected && <span style={{ fontWeight: 900 }}>✓</span>}
                      </button>
                    );
                  })}
                </div>
              </div>

              {selectedMethod === "qris" && (
                <div
                  style={{
                    display: "flex",
                    flexDirection: "column",
                    alignItems: "center",
                    padding: "1rem",
                    border: "2px dashed #1b1b2f",
                    borderRadius: "0.75rem",
                    backgroundColor: "#ffffff",
                    textAlign: "center",
                  }}
                >
                  <div
                    style={{
                      width: "160px",
                      height: "160px",
                      backgroundColor: "#f5f5f5",
                      border: "3px solid #1b1b2f",
                      borderRadius: "0.5rem",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      fontWeight: 900,
                      fontSize: "0.875rem",
                      boxShadow: "3px 3px 0 #1b1b2f",
                      marginBottom: "0.5rem",
                    }}
                  >
                    [ QRIS KALENGO ]
                  </div>
                  <span style={{ fontSize: "0.75rem", color: "#666", fontWeight: 700 }}>
                    Scan menggunakan aplikasi m-Banking atau e-Wallet favoritmu
                  </span>
                </div>
              )}
            </div>

            <footer className="modal-footer" style={{ justifyContent: "space-between", alignItems: "center" }}>
              <button
                type="button"
                className="btn btn--secondary"
                onClick={onClose}
                disabled={isProcessing}
              >
                Tutup
              </button>

              <button
                type="button"
                className="btn btn--mint"
                onClick={handleSimulatePayment}
                disabled={isProcessing}
                style={{ fontWeight: 900 }}
              >
                {isProcessing ? "Memproses Verifikasi..." : "Simulasikan Bayar Sukses ✓"}
              </button>
            </footer>
          </div>
        )}
      </div>
    </div>
  );
};
