import React, { useEffect, useState } from "react";
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

interface PaymentMethod {
  id: "qris" | "bca_va" | "mandiri_va" | "bni_va" | "gopay";
  name: string;
  category: "qris" | "va" | "ewallet";
  logo: string;
  desc: string;
  vaNumber?: string;
  bankName?: string;
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
  const [progressPercent, setProgressPercent] = useState<number>(0);
  const [copiedVa, setCopiedVa] = useState<boolean>(false);
  const [timeLeft, setTimeLeft] = useState<number>(15 * 60); // 15 menit
  const [showGopayQr, setShowGopayQr] = useState<boolean>(false);
  const toast = useToast();

  const paymentMethods: PaymentMethod[] = [
    {
      id: "qris",
      name: "QRIS Instant",
      category: "qris",
      logo: "📱",
      desc: "BCA, GoPay, OVO, ShopeePay, Dana, LinkAja",
    },
    {
      id: "bca_va",
      name: "BCA Virtual Account",
      category: "va",
      logo: "🏦",
      desc: "No. VA: 8271 0812 9923 1184",
      vaNumber: "8271 0812 9923 1184",
      bankName: "Bank Central Asia (BCA)",
    },
    {
      id: "mandiri_va",
      name: "Mandiri Virtual Account",
      category: "va",
      logo: "💳",
      desc: "No. VA: 8870 0812 9923 1184",
      vaNumber: "8870 0812 9923 1184",
      bankName: "Bank Mandiri",
    },
    {
      id: "bni_va",
      name: "BNI Virtual Account",
      category: "va",
      logo: "🏛️",
      desc: "No. VA: 9881 0812 9923 1184",
      vaNumber: "9881 0812 9923 1184",
      bankName: "Bank Negara Indonesia (BNI)",
    },
    {
      id: "gopay",
      name: "GoPay / E-Wallet",
      category: "ewallet",
      logo: "🟢",
      desc: "Notifikasi ponsel langsung & saldo GoPay",
    },
  ];

  // Hitung mundur waktu pembayaran 15:00
  useEffect(() => {
    if (!isOpen || isPaid) return;
    const interval = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          clearInterval(interval);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(interval);
  }, [isOpen, isPaid]);

  if (!isOpen || !order) return null;

  const currentMethod =
    paymentMethods.find((m) => m.id === selectedMethod) ?? paymentMethods[0]!;

  const formatCountdown = (seconds: number) => {
    const m = Math.floor(seconds / 60)
      .toString()
      .padStart(2, "0");
    const s = (seconds % 60).toString().padStart(2, "0");
    return `${m}:${s}`;
  };

  const handleCopyVa = async (vaNumber: string) => {
    const rawNumber = vaNumber.replace(/\s+/g, "");
    try {
      if (typeof navigator !== "undefined" && navigator.clipboard) {
        await navigator.clipboard.writeText(rawNumber);
      } else {
        const textarea = document.createElement("textarea");
        textarea.value = rawNumber;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand("copy");
        document.body.removeChild(textarea);
      }
    } catch {
      // Fallback
    }
    setCopiedVa(true);
    toast.success("✓ Nomor Virtual Account berhasil disalin!", "Berhasil Disalin");
    setTimeout(() => setCopiedVa(false), 2000);
  };

  const handleDownloadReceipt = () => {
    const courierInfo = order.courier
      ? `${order.courier.toUpperCase()} ${order.courier_service || ""}`.trim()
      : "Kurir Standar";

    const itemsList =
      order.items && order.items.length > 0
        ? order.items
            .map(
              (i) =>
                `• ${i.product_name} (${i.pack_size ? `${i.pack_size}ml` : "Standard"}) x${i.quantity} : ${formatIdr(i.line_total)}`,
            )
            .join("\n")
        : `• Total Minuman : ${formatIdr(order.subtotal)}`;

    const receiptText = `================================================
          STRUK PEMBAYARAN RESMI
             KALENGO BEVERAGES
================================================
Nomor Pesanan  : ${order.order_no}
Waktu Transaksi: ${new Date().toLocaleString("id-ID")}
Status         : LUNAS (SETTLEMENT)
Metode Bayar   : ${currentMethod.name}
------------------------------------------------
RINCIAN PESANAN:
${itemsList}
------------------------------------------------
Subtotal       : ${formatIdr(order.subtotal)}
Ongkos Kirim   : ${formatIdr(order.shipping_cost)} (${courierInfo})
Diskon Voucher : ${order.discount > 0 ? `-${formatIdr(order.discount)}` : "Rp0"}
TOTAL BAYAR    : ${formatIdr(order.total)}
================================================
Terima kasih telah berbelanja di KALENGO!
Minuman dingin segar langsung siap diantar.
Customer Care: halo@kalengo.co.id
================================================`;

    try {
      const blob = new Blob([receiptText], { type: "text/plain;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `struk-kalengo-${order.order_no}.txt`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch {
      // Fallback
    }
    toast.success("✓ Bukti pembayaran berhasil diunduh!", "Unduh Berhasil");
  };

  const handleSimulatePayment = async () => {
    setIsProcessing(true);
    setProgressPercent(15);

    // Animasi progress bar simulasi Midtrans
    const progressInterval = setInterval(() => {
      setProgressPercent((prev) => {
        if (prev >= 90) {
          clearInterval(progressInterval);
          return 90;
        }
        return prev + 25;
      });
    }, 200);

    // Kirim webhook settlement ke backend jika ini pesanan server nyata
    if (
      order.order_no.startsWith("KLG-") &&
      typeof crypto !== "undefined" &&
      crypto.subtle
    ) {
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
      clearInterval(progressInterval);
      setProgressPercent(100);
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
    }, 1100);
  };

  return (
    <div
      className="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="payment-modal-title"
    >
      <div
        className="modal-window"
        style={{ maxWidth: "560px", position: "relative", overflow: "hidden" }}
      >
        {isPaid ? (
          // Layar Bukti Pembayaran Berhasil (Success Receipt)
          <div style={{ position: "relative" }}>
            <CelebrationConfetti active={isPaid} />

            <header
              className="modal-header"
              style={{ backgroundColor: "var(--mint, #3ddc97)" }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                <span style={{ fontSize: "1.5rem" }} aria-hidden="true">
                  🎉
                </span>
                <h2 id="payment-modal-title" className="modal-title">
                  Pembayaran Berhasil!
                </h2>
              </div>
              <button
                type="button"
                className="btn btn--sm btn--secondary"
                onClick={onClose}
                aria-label="Tutup bukti pembayaran"
                style={{ padding: "0.35rem 0.65rem" }}
              >
                ✕
              </button>
            </header>

            <div
              className="modal-body"
              style={{
                textAlign: "center",
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                gap: "1rem",
              }}
            >
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  gap: "1rem",
                  margin: "0.5rem 0",
                }}
              >
                <div className="bouncy-checkmark-circle" aria-hidden="true">
                  ✓
                </div>
                <div className="mascot-anim--cheer">
                  <CanSodaLemon size={130} alt="Maskot Kalengo Merayakan Sukses" />
                </div>
              </div>

              <div style={{ display: "flex", flexDirection: "column", gap: "0.25rem" }}>
                <h3
                  style={{
                    fontFamily: "var(--font-display)",
                    fontSize: "1.375rem",
                    margin: 0,
                  }}
                >
                  Hore! Minumanmu Segera Dikirim 🚚
                </h3>
                <span
                  className="badge badge--mint"
                  style={{
                    alignSelf: "center",
                    fontWeight: 900,
                    fontSize: "0.8125rem",
                    letterSpacing: "0.05em",
                  }}
                >
                  LUNAS (SETTLEMENT)
                </span>
              </div>

              <p
                style={{
                  color: "#4a4a63",
                  fontSize: "0.875rem",
                  margin: 0,
                  maxWidth: "440px",
                }}
              >
                Pesanan nomor <strong>{order.order_no}</strong> berhasil
                diselesaikan melalui <strong>{currentMethod.name}</strong> dan siap
                dikemas dingin di gudang KALENGO.
              </p>

              {/* Rincian Struk Resmi */}
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
                <div
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    marginBottom: "0.35rem",
                  }}
                >
                  <span style={{ color: "#666" }}>Nomor Pesanan</span>
                  <span style={{ fontWeight: 800 }}>{order.order_no}</span>
                </div>
                <div
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    marginBottom: "0.35rem",
                  }}
                >
                  <span style={{ color: "#666" }}>Waktu Transaksi</span>
                  <span style={{ fontWeight: 800 }}>
                    {new Date().toLocaleTimeString("id-ID", {
                      hour: "2-digit",
                      minute: "2-digit",
                    })}{" "}
                    WIB
                  </span>
                </div>
                <div
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    marginBottom: "0.35rem",
                  }}
                >
                  <span style={{ color: "#666" }}>Metode Bayar</span>
                  <span style={{ fontWeight: 800 }}>{currentMethod.name}</span>
                </div>
                <div
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    marginBottom: "0.35rem",
                  }}
                >
                  <span style={{ color: "#666" }}>Total Pembayaran</span>
                  <span style={{ fontWeight: 900, fontSize: "1.0625rem" }}>
                    {formatIdr(order.total)}
                  </span>
                </div>
                <div
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    borderTop: "2px dashed #1b1b2f",
                    paddingTop: "0.4rem",
                    marginTop: "0.4rem",
                  }}
                >
                  <span style={{ color: "#666" }}>Status Transaksi</span>
                  <span
                    style={{
                      fontWeight: 900,
                      color: "var(--mint-dark, #0f6840)",
                    }}
                  >
                    ✓ SETTLEMENT (LUNAS)
                  </span>
                </div>
              </div>
            </div>

            <footer
              className="modal-footer"
              style={{
                justifyContent: "center",
                gap: "0.75rem",
                flexWrap: "wrap",
              }}
            >
              <button
                type="button"
                className="btn btn--secondary"
                onClick={handleDownloadReceipt}
              >
                Unduh Bukti Transaksi 📄
              </button>
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
          // Layar Pilihan Pembayaran & Instruksi (Midtrans Snap Simulation)
          <div>
            <header
              className="modal-header"
              style={{ backgroundColor: "var(--lemon, #ffd23f)" }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                <span style={{ fontSize: "1.5rem" }} aria-hidden="true">
                  💳
                </span>
                <div>
                  <h2
                    id="payment-modal-title"
                    className="modal-title"
                    style={{ fontSize: "1.25rem" }}
                  >
                    Bayar Pesanan
                  </h2>
                  <span
                    style={{
                      fontSize: "0.75rem",
                      fontWeight: 700,
                      color: "#4a4a63",
                    }}
                  >
                    No: {order.order_no}
                  </span>
                </div>
              </div>
              <button
                type="button"
                className="btn btn--sm btn--secondary"
                onClick={onClose}
                disabled={isProcessing}
                aria-label="Tutup pembayaran"
                style={{ padding: "0.35rem 0.65rem" }}
              >
                ✕
              </button>
            </header>

            <div
              className="modal-body"
              style={{ display: "flex", flexDirection: "column", gap: "1rem" }}
            >
              {/* Box Total Tagihan & Countdown */}
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  padding: "0.875rem 1rem",
                  backgroundColor: "var(--cream, #fff4dc)",
                  border: "2px solid #1b1b2f",
                  borderRadius: "0.5rem",
                }}
              >
                <div>
                  <div
                    style={{
                      fontSize: "0.75rem",
                      color: "#6a6a80",
                      fontWeight: 700,
                    }}
                  >
                    Total Tagihan
                  </div>
                  <div
                    style={{
                      fontSize: "1.5rem",
                      fontWeight: 900,
                      color: "var(--ink, #1b1b2f)",
                    }}
                  >
                    {formatIdr(order.total)}
                  </div>
                </div>
                <div style={{ textAlign: "right" }}>
                  <span className="badge badge--hot" style={{ fontSize: "0.75rem" }}>
                    ⏱ Sisa Waktu {formatCountdown(timeLeft)}
                  </span>
                </div>
              </div>

              {/* Progress bar jika sedang proses verifikasi */}
              {isProcessing && (
                <div
                  role="status"
                  aria-live="polite"
                  style={{
                    backgroundColor: "#f2fcf6",
                    border: "2.5px solid var(--mint, #3ddc97)",
                    borderRadius: "0.75rem",
                    padding: "1rem",
                    textAlign: "center",
                    display: "flex",
                    flexDirection: "column",
                    gap: "0.5rem",
                  }}
                >
                  <div style={{ fontWeight: 800, fontSize: "0.875rem" }}>
                    🔄 Memverifikasi transaksi dengan Midtrans...
                  </div>
                  <div className="payment-progress-bar">
                    <div
                      className="payment-progress-fill"
                      style={{ width: `${progressPercent}%` }}
                    />
                  </div>
                  <div style={{ fontSize: "0.75rem", color: "#666" }}>
                    Menghubungi gateway pembayaran & mengonfirmasi dana...
                  </div>
                </div>
              )}

              {/* Pilihan Metode Pembayaran */}
              <div>
                <label
                  style={{
                    display: "block",
                    fontSize: "0.8125rem",
                    fontWeight: 800,
                    marginBottom: "0.5rem",
                  }}
                >
                  Pilih Kanal Pembayaran (Midtrans Snap):
                </label>
                <div
                  style={{
                    display: "flex",
                    flexDirection: "column",
                    gap: "0.5rem",
                  }}
                >
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
                          padding: "0.65rem 0.85rem",
                          textAlign: "left",
                        }}
                        onClick={() => setSelectedMethod(pm.id)}
                        disabled={isProcessing}
                      >
                        <span style={{ fontSize: "1.25rem" }} aria-hidden="true">
                          {pm.logo}
                        </span>
                        <div style={{ flex: 1 }}>
                          <div style={{ fontWeight: 900, fontSize: "0.875rem" }}>
                            {pm.name}
                          </div>
                          <div
                            style={{
                              fontSize: "0.7rem",
                              color: "#666",
                              fontWeight: 600,
                            }}
                          >
                            {pm.desc}
                          </div>
                        </div>
                        {isSelected && (
                          <span style={{ fontWeight: 900, color: "#147d53" }}>✓</span>
                        )}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* A) Tampilan QRIS Instant */}
              {selectedMethod === "qris" && (
                <div className="qris-frame">
                  <div
                    style={{
                      width: "100%",
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                      borderBottom: "2px solid #1b1b2f",
                      paddingBottom: "0.5rem",
                      marginBottom: "0.75rem",
                    }}
                  >
                    <div style={{ fontWeight: 900, fontSize: "0.875rem", color: "#b91c1c" }}>
                      QRIS • STANDAR NASIONAL
                    </div>
                    <span className="badge badge--sky" style={{ fontSize: "0.7rem" }}>
                      GPN / ASPI
                    </span>
                  </div>

                  {/* SVG QRIS Canvas */}
                  <div style={{ position: "relative", display: "inline-block" }}>
                    <svg
                      width="180"
                      height="180"
                      viewBox="0 0 180 180"
                      style={{
                        backgroundColor: "#ffffff",
                        border: "3px solid #1b1b2f",
                        borderRadius: "0.5rem",
                      }}
                      role="img"
                      aria-label="QRIS Code KALENGO"
                    >
                      {/* Corner Finder Patterns */}
                      {/* Top Left */}
                      <rect x="15" y="15" width="40" height="40" rx="4" fill="#1b1b2f" />
                      <rect x="22" y="22" width="26" height="26" rx="2" fill="#ffffff" />
                      <rect x="28" y="28" width="14" height="14" rx="2" fill="#1b1b2f" />

                      {/* Top Right */}
                      <rect x="125" y="15" width="40" height="40" rx="4" fill="#1b1b2f" />
                      <rect x="132" y="22" width="26" height="26" rx="2" fill="#ffffff" />
                      <rect x="138" y="28" width="14" height="14" rx="2" fill="#1b1b2f" />

                      {/* Bottom Left */}
                      <rect x="15" y="125" width="40" height="40" rx="4" fill="#1b1b2f" />
                      <rect x="22" y="132" width="26" height="26" rx="2" fill="#ffffff" />
                      <rect x="28" y="138" width="14" height="14" rx="2" fill="#1b1b2f" />

                      {/* QR Matrix Grid Dots */}
                      <g fill="#1b1b2f">
                        {/* Baris data simulasi */}
                        <rect x="65" y="18" width="8" height="8" />
                        <rect x="80" y="18" width="8" height="8" />
                        <rect x="95" y="18" width="8" height="8" />
                        <rect x="108" y="18" width="8" height="8" />

                        <rect x="65" y="32" width="8" height="8" />
                        <rect x="85" y="32" width="14" height="8" />
                        <rect x="105" y="32" width="8" height="8" />

                        <rect x="65" y="46" width="12" height="8" />
                        <rect x="85" y="46" width="8" height="8" />
                        <rect x="100" y="46" width="12" height="8" />

                        <rect x="18" y="65" width="8" height="8" />
                        <rect x="32" y="65" width="14" height="8" />
                        <rect x="52" y="65" width="8" height="8" />
                        <rect x="120" y="65" width="10" height="8" />
                        <rect x="138" y="65" width="8" height="8" />
                        <rect x="152" y="65" width="12" height="8" />

                        <rect x="18" y="80" width="12" height="8" />
                        <rect x="36" y="80" width="8" height="8" />
                        <rect x="125" y="80" width="12" height="8" />
                        <rect x="145" y="80" width="14" height="8" />

                        <rect x="18" y="95" width="8" height="8" />
                        <rect x="32" y="95" width="12" height="8" />
                        <rect x="50" y="95" width="8" height="8" />
                        <rect x="122" y="95" width="8" height="8" />
                        <rect x="138" y="95" width="12" height="8" />
                        <rect x="156" y="95" width="8" height="8" />

                        <rect x="65" y="125" width="8" height="8" />
                        <rect x="80" y="125" width="12" height="8" />
                        <rect x="100" y="125" width="8" height="8" />
                        <rect x="115" y="125" width="10" height="8" />
                        <rect x="135" y="125" width="8" height="8" />
                        <rect x="150" y="125" width="14" height="8" />

                        <rect x="65" y="140" width="14" height="8" />
                        <rect x="88" y="140" width="8" height="8" />
                        <rect x="102" y="140" width="12" height="8" />
                        <rect x="125" y="140" width="14" height="8" />
                        <rect x="145" y="140" width="8" height="8" />

                        <rect x="65" y="155" width="8" height="8" />
                        <rect x="80" y="155" width="10" height="8" />
                        <rect x="100" y="155" width="12" height="8" />
                        <rect x="120" y="155" width="8" height="8" />
                        <rect x="140" y="155" width="12" height="8" />
                      </g>
                    </svg>

                    {/* Logo KALENGO di Pusat QR */}
                    <div className="qris-badge-center">
                      🥤 KALENGO
                    </div>
                  </div>

                  <div style={{ marginTop: "0.5rem", fontSize: "0.75rem", fontWeight: 800 }}>
                    NMID: ID1020039281729 • KALENGO BEVERAGES
                  </div>

                  {/* Instruksi Langkah QRIS */}
                  <div
                    style={{
                      width: "100%",
                      marginTop: "0.75rem",
                      padding: "0.75rem",
                      backgroundColor: "var(--cream, #fff4dc)",
                      border: "1.5px solid var(--ink, #1b1b2f)",
                      borderRadius: "0.5rem",
                      textAlign: "left",
                      fontSize: "0.75rem",
                    }}
                  >
                    <div style={{ fontWeight: 800, marginBottom: "0.25rem" }}>
                      Cara Pembayaran QRIS:
                    </div>
                    <ol style={{ margin: 0, paddingLeft: "1.25rem", lineHeight: 1.5 }}>
                      <li>Buka aplikasi m-Banking (BCA, Mandiri, BRI) atau e-Wallet (GoPay, OVO, ShopeePay, Dana).</li>
                      <li>Pilih menu <strong>Bayar / Scan QRIS</strong>.</li>
                      <li>Arahkan kamera ke kode QRIS di atas.</li>
                      <li>Periksa nama penerima <strong>KALENGO BEVERAGES</strong> dan total bayar.</li>
                      <li>Masukkan PIN Anda lalu klik tombol <strong>Konfirmasi Bayar</strong> di bawah.</li>
                    </ol>
                  </div>
                </div>
              )}

              {/* B) Tampilan Virtual Account (BCA / Mandiri / BNI) */}
              {currentMethod.category === "va" && currentMethod.vaNumber && (
                <div
                  style={{
                    display: "flex",
                    flexDirection: "column",
                    gap: "0.75rem",
                    padding: "1rem",
                    border: "2.5px solid var(--ink, #1b1b2f)",
                    borderRadius: "0.75rem",
                    backgroundColor: "#ffffff",
                    boxShadow: "3px 3px 0 var(--ink, #1b1b2f)",
                  }}
                >
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <span style={{ fontWeight: 800, fontSize: "0.8125rem", color: "#666" }}>
                      {currentMethod.bankName}
                    </span>
                    <span className="badge badge--mint" style={{ fontSize: "0.7rem" }}>
                      Otomatis
                    </span>
                  </div>

                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "space-between",
                      padding: "0.75rem 1rem",
                      backgroundColor: "var(--cream, #fff4dc)",
                      border: "2px solid var(--ink, #1b1b2f)",
                      borderRadius: "0.5rem",
                    }}
                  >
                    <div>
                      <div style={{ fontSize: "0.7rem", color: "#6a6a80", fontWeight: 700 }}>
                        Nomor Virtual Account:
                      </div>
                      <div
                        style={{
                          fontSize: "1.25rem",
                          fontWeight: 900,
                          letterSpacing: "0.05em",
                          fontFamily: "monospace",
                          color: "var(--ink, #1b1b2f)",
                        }}
                      >
                        {currentMethod.vaNumber}
                      </div>
                    </div>

                    <button
                      type="button"
                      className={`btn-copy ${copiedVa ? "btn-copy--copied" : ""}`}
                      onClick={() => handleCopyVa(currentMethod.vaNumber!)}
                      aria-label="Salin Nomor Virtual Account"
                    >
                      {copiedVa ? "✓ Tersalin!" : "📋 Salin No. VA"}
                    </button>
                  </div>

                  {/* Instruksi Transfer VA */}
                  <div
                    style={{
                      padding: "0.75rem",
                      backgroundColor: "#f9f9fb",
                      border: "1.5px solid #1b1b2f",
                      borderRadius: "0.5rem",
                      fontSize: "0.75rem",
                    }}
                  >
                    <div style={{ fontWeight: 800, marginBottom: "0.25rem" }}>
                      Panduan Transfer Virtual Account:
                    </div>
                    <ol style={{ margin: 0, paddingLeft: "1.25rem", lineHeight: 1.5 }}>
                      <li>Buka m-Banking atau ATM {currentMethod.bankName}.</li>
                      <li>Pilih menu <strong>Transfer &gt; Virtual Account</strong>.</li>
                      <li>Masukkan 16-digit nomor VA di atas.</li>
                      <li>Pastikan total bayar <strong>{formatIdr(order.total)}</strong> dan nama merchant <strong>KALENGO</strong>.</li>
                      <li>Selesaikan pembayaran lalu klik <strong>Konfirmasi Bayar</strong> di bawah.</li>
                    </ol>
                  </div>
                </div>
              )}

              {/* C) Tampilan GoPay / E-Wallet */}
              {selectedMethod === "gopay" && (
                <div
                  style={{
                    display: "flex",
                    flexDirection: "column",
                    gap: "0.75rem",
                    padding: "1rem",
                    border: "2.5px solid var(--ink, #1b1b2f)",
                    borderRadius: "0.75rem",
                    backgroundColor: "#ffffff",
                    boxShadow: "3px 3px 0 var(--ink, #1b1b2f)",
                  }}
                >
                  {/* Simulasi Push Notification Handphone */}
                  <div
                    style={{
                      display: "flex",
                      alignItems: "flex-start",
                      gap: "0.75rem",
                      padding: "0.75rem 1rem",
                      backgroundColor: "#e6f9f0",
                      border: "2px solid #3ddc97",
                      borderRadius: "0.5rem",
                    }}
                  >
                    <span style={{ fontSize: "1.5rem" }} aria-hidden="true">
                      📲
                    </span>
                    <div style={{ flex: 1, fontSize: "0.8125rem" }}>
                      <div style={{ fontWeight: 900, color: "#0f6840" }}>
                        Notifikasi Terkirim ke Ponsel Anda
                      </div>
                      <div style={{ color: "#222", marginTop: "0.15rem" }}>
                        Permintaan bayar senilai <strong>{formatIdr(order.total)}</strong> telah dikirim ke aplikasi Gojek Anda.
                      </div>
                    </div>
                  </div>

                  <div style={{ display: "flex", gap: "0.5rem", flexWrap: "wrap" }}>
                    <button
                      type="button"
                      className="btn btn--sm btn--lemon"
                      onClick={() => {
                        toast.success(
                          "Membuka aplikasi Gojek di perangkat seluler...",
                          "Buka Gojek",
                        );
                      }}
                    >
                      Buka Aplikasi Gojek ↗
                    </button>
                    <button
                      type="button"
                      className="btn btn--sm btn--secondary"
                      onClick={() => setShowGopayQr(!showGopayQr)}
                    >
                      {showGopayQr ? "Sembunyikan QR GoPay" : "Tampilkan QR GoPay"}
                    </button>
                  </div>

                  {showGopayQr && (
                    <div
                      style={{
                        display: "flex",
                        flexDirection: "column",
                        alignItems: "center",
                        padding: "0.75rem",
                        backgroundColor: "var(--cream, #fff4dc)",
                        border: "1.5px dashed var(--ink, #1b1b2f)",
                        borderRadius: "0.5rem",
                      }}
                    >
                      <div
                        style={{
                          width: "120px",
                          height: "120px",
                          backgroundColor: "#fff",
                          border: "2px solid #1b1b2f",
                          borderRadius: "0.5rem",
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          fontWeight: 900,
                          fontSize: "0.75rem",
                        }}
                      >
                        [ QR GOPAY ]
                      </div>
                      <span style={{ fontSize: "0.7rem", marginTop: "0.25rem", fontWeight: 700 }}>
                        Scan langsung menggunakan scanner Gojek
                      </span>
                    </div>
                  )}
                </div>
              )}
            </div>

            <footer
              className="modal-footer"
              style={{
                justifyContent: "space-between",
                alignItems: "center",
              }}
            >
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
                {isProcessing ? "Memproses Verifikasi..." : "Konfirmasi Bayar 👉"}
              </button>
            </footer>
          </div>
        )}
      </div>
    </div>
  );
};
