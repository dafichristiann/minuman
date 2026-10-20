import React, { useEffect, useRef, useState } from "react";
import { useToast } from "../context/useToast";
import { CelebrationConfetti } from "./CelebrationConfetti";

export interface WhatsAppVerifyModalProps {
  isOpen: boolean;
  phone: string;
  onClose: () => void;
  onVerified: (verifiedPhone: string) => void;
}

export const WhatsAppVerifyModal: React.FC<WhatsAppVerifyModalProps> = ({
  isOpen,
  phone,
  onClose,
  onVerified,
}) => {
  const toast = useToast();
  const inputRef = useRef<HTMLInputElement>(null);

  const [simulatedOtp, setSimulatedOtp] = useState<string>("7294");
  const [otp, setOtp] = useState<string>("");
  const [countdown, setCountdown] = useState<number>(60);
  const [isShaking, setIsShaking] = useState<boolean>(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isSuccess, setIsSuccess] = useState<boolean>(false);

  // Generate new 4-digit code
  const generateNewCode = () => {
    return Math.floor(1000 + Math.random() * 9000).toString();
  };

  // Reset state when modal opens
  useEffect(() => {
    if (!isOpen) return;

    const initialCode = generateNewCode();
    setSimulatedOtp(initialCode);
    setOtp("");
    setCountdown(60);
    setIsShaking(false);
    setErrorMessage(null);
    setIsSuccess(false);

    // Auto focus input
    const timer = setTimeout(() => {
      inputRef.current?.focus();
    }, 50);
    return () => clearTimeout(timer);
  }, [isOpen, phone]);

  // Resend countdown timer
  useEffect(() => {
    if (!isOpen || countdown <= 0 || isSuccess) return;

    const interval = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          clearInterval(interval);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [isOpen, countdown, isSuccess]);

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen && !isSuccess) {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, isSuccess, onClose]);

  if (!isOpen) return null;

  const handleResendOtp = () => {
    if (countdown > 0) return;
    const newCode = generateNewCode();
    setSimulatedOtp(newCode);
    setOtp("");
    setErrorMessage(null);
    setCountdown(60);
    toast.info("Kode OTP WhatsApp baru telah dikirimkan!", "WhatsApp OTP");
  };

  const handleFillCode = () => {
    setOtp(simulatedOtp);
    setErrorMessage(null);
    inputRef.current?.focus();
  };

  const handleVerify = (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (otp.length < 4 || isSuccess) return;

    if (otp.trim() !== simulatedOtp) {
      setIsShaking(true);
      setTimeout(() => setIsShaking(false), 500);
      const errorText = "✕ Kode OTP salah. Silakan periksa kembali pesan WhatsApp Anda.";
      setErrorMessage(errorText);
      toast.error(errorText, "Verifikasi Gagal");
      return;
    }

    // Match success
    setIsSuccess(true);
    setErrorMessage(null);
    toast.success(
      `✓ Nomor WhatsApp ${phone} berhasil diverifikasi! 🛡️`,
      "Verifikasi Berhasil",
    );

    setTimeout(() => {
      onVerified(phone);
      onClose();
    }, 400);
  };

  return (
    <div
      className="modal-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget && !isSuccess) onClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="whatsapp-verify-title"
      style={{ zIndex: 1200 }}
    >
      <div
        className="modal-window"
        style={{
          maxWidth: "460px",
          position: "relative",
          overflow: "hidden",
        }}
      >
        {isSuccess && <CelebrationConfetti active={true} />}

        {/* Modal Header */}
        <header
          className="modal-header"
          style={{
            backgroundColor: "#25D366",
            color: "#ffffff",
            borderBottom: "3px solid var(--ink, #1b1b2f)",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ fontSize: "1.4rem" }} aria-hidden="true">
              💬
            </span>
            <h2
              id="whatsapp-verify-title"
              className="modal-title"
              style={{
                fontSize: "1.125rem",
                fontWeight: 900,
                color: "#ffffff",
                letterSpacing: "-0.01em",
              }}
            >
              Verifikasi Nomor WhatsApp 💬
            </h2>
          </div>
          <button
            type="button"
            className="btn btn--sm btn--secondary"
            onClick={onClose}
            disabled={isSuccess}
            aria-label="Tutup verifikasi WhatsApp"
            style={{
              padding: "0.25rem 0.5rem",
              minHeight: "32px",
              minWidth: "32px",
              lineHeight: 1,
            }}
          >
            ✕
          </button>
        </header>

        {/* Modal Body */}
        <div className="modal-body" style={{ padding: "1.25rem", display: "flex", flexDirection: "column", gap: "1rem" }}>
          {/* Explanatory note */}
          <p
            style={{
              margin: 0,
              fontSize: "0.875rem",
              color: "var(--ink, #1b1b2f)",
              lineHeight: 1.5,
              fontWeight: 600,
            }}
          >
            Kami telah mengirimkan 4-digit kode OTP ke nomor WhatsApp{" "}
            <strong style={{ fontWeight: 900, color: "#128C7E" }}>{phone}</strong>{" "}
            untuk validasi pengiriman kurir.
          </p>

          {/* Simulation Banner */}
          <div
            style={{
              backgroundColor: "var(--cream, #fff4dc)",
              border: "2px dashed var(--ink, #1b1b2f)",
              borderRadius: "var(--radius-sm, 0.5rem)",
              padding: "0.75rem 0.875rem",
              display: "flex",
              flexDirection: "column",
              gap: "0.5rem",
            }}
          >
            <div
              style={{
                fontSize: "0.8125rem",
                fontWeight: 700,
                color: "var(--ink, #1b1b2f)",
                display: "flex",
                alignItems: "center",
                flexWrap: "wrap",
                gap: "0.25rem",
              }}
            >
              <span>💡 [Simulasi WhatsApp Gateway]: Kode OTP Anda adalah</span>
              <span
                style={{
                  backgroundColor: "#25D366",
                  color: "#ffffff",
                  padding: "0.1rem 0.45rem",
                  borderRadius: "4px",
                  fontWeight: 900,
                  letterSpacing: "0.1em",
                  border: "1.5px solid var(--ink, #1b1b2f)",
                }}
              >
                {simulatedOtp}
              </span>
            </div>
            <div>
              <button
                type="button"
                onClick={handleFillCode}
                className="btn btn--sm btn--lemon"
                style={{
                  fontSize: "0.75rem",
                  padding: "0.3rem 0.65rem",
                  minHeight: "30px",
                  fontWeight: 800,
                }}
              >
                Gunakan Kode Ini
              </button>
            </div>
          </div>

          {/* OTP Input Form */}
          <form onSubmit={handleVerify} style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
            <label
              htmlFor="whatsapp-otp-input"
              style={{
                display: "block",
                fontSize: "0.8125rem",
                fontWeight: 800,
                color: "var(--ink, #1b1b2f)",
                textAlign: "center",
              }}
            >
              Masukkan 4-Digit Kode OTP:
            </label>

            <div style={{ display: "flex", justifyContent: "center" }}>
              <input
                ref={inputRef}
                id="whatsapp-otp-input"
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                maxLength={4}
                autoComplete="one-time-code"
                value={otp}
                aria-invalid={errorMessage ? true : undefined}
                aria-describedby={errorMessage ? "whatsapp-otp-error" : undefined}
                onChange={(e) => {
                  const val = e.target.value.replace(/\D/g, "").slice(0, 4);
                  setOtp(val);
                  if (errorMessage) setErrorMessage(null);
                }}
                placeholder="••••"
                className={`${isShaking ? "input--shake input--invalid" : ""} ${
                  isSuccess ? "input--valid" : ""
                }`}
                style={{
                  width: "180px",
                  fontSize: "1.5rem",
                  letterSpacing: "0.5rem",
                  textAlign: "center",
                  fontWeight: 900,
                  padding: "0.5rem 0.75rem",
                  border: isSuccess
                    ? "2.5px solid var(--mint, #3ddc97)"
                    : errorMessage
                      ? "2.5px solid var(--tomato, #ff4b3e)"
                      : "2.5px solid var(--ink, #1b1b2f)",
                  borderRadius: "var(--radius-sm, 0.5rem)",
                  backgroundColor: isSuccess ? "#f2fcf6" : "#ffffff",
                  outline: "none",
                  boxShadow: "3px 3px 0 var(--ink, #1b1b2f)",
                }}
              />
            </div>

            {/* Error Badge */}
            {errorMessage && (
              <div
                id="whatsapp-otp-error"
                role="alert"
                style={{
                  backgroundColor: "#ffe0dc",
                  border: "2px solid var(--tomato, #ff4b3e)",
                  borderRadius: "var(--radius-sm, 0.5rem)",
                  padding: "0.5rem 0.75rem",
                  color: "#b91c1c",
                  fontWeight: 800,
                  fontSize: "0.8125rem",
                  textAlign: "center",
                }}
              >
                {errorMessage}
              </div>
            )}

            {/* Success Badge */}
            {isSuccess && (
              <div
                role="status"
                style={{
                  backgroundColor: "#e6fbf1",
                  border: "2px solid var(--mint, #3ddc97)",
                  borderRadius: "var(--radius-sm, 0.5rem)",
                  padding: "0.5rem 0.75rem",
                  color: "#0f766e",
                  fontWeight: 900,
                  fontSize: "0.875rem",
                  textAlign: "center",
                }}
              >
                ✓ Verifikasi Berhasil! Menyiapkan checkout...
              </div>
            )}

            {/* Resend OTP Timer & Button */}
            <div style={{ textAlign: "center", marginTop: "0.25rem" }}>
              <button
                type="button"
                onClick={handleResendOtp}
                disabled={countdown > 0 || isSuccess}
                style={{
                  background: "none",
                  border: "none",
                  color: countdown > 0 ? "var(--ink-soft, #666)" : "#128C7E",
                  cursor: countdown > 0 ? "not-allowed" : "pointer",
                  fontWeight: 800,
                  fontSize: "0.8125rem",
                  textDecoration: countdown === 0 ? "underline" : "none",
                }}
              >
                {countdown > 0
                  ? `Kirim Ulang Kode (${countdown}s)`
                  : "Kirim Ulang Kode"}
              </button>
            </div>

            {/* Actions */}
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "1fr 1fr",
                gap: "0.75rem",
                marginTop: "0.75rem",
              }}
            >
              <button
                type="button"
                className="btn btn--secondary"
                onClick={onClose}
                disabled={isSuccess}
                style={{
                  fontWeight: 800,
                  fontSize: "0.875rem",
                }}
              >
                Batal / Ubah Nomor
              </button>
              <button
                type="submit"
                className="btn btn--whatsapp"
                disabled={otp.length < 4 || isSuccess}
                style={{
                  fontWeight: 900,
                  fontSize: "0.875rem",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  gap: "0.35rem",
                }}
              >
                {isSuccess ? "✓ Berhasil!" : "Verifikasi Sekarang"}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};
