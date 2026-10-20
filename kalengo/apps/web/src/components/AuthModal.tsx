import React, { useEffect, useState } from "react";
import { ApiError } from "../api/client";
import { useAuth } from "../context/AuthContext";
import { useToast } from "../context/useToast";

export interface AuthModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: () => void;
  initialMode?: "login" | "register";
}

export const AuthModal: React.FC<AuthModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
  initialMode = "login",
}) => {
  const { login, register } = useAuth();
  const toast = useToast();
  const [mode, setMode] = useState<"login" | "register">(initialMode);

  // Form states
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [phone, setPhone] = useState("");
  const [acceptTerms, setAcceptTerms] = useState(true);

  // Status states
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    setMode(initialMode);
    setErrorMessage(null);
  }, [initialMode, isOpen]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen && !isLoading) {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, isLoading, onClose]);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setErrorMessage(null);

    try {
      if (mode === "login") {
        await login({ email, password });
        toast.success("Selamat datang kembali di KALENGO!", "Berhasil Masuk");
      } else {
        if (password.length < 10) {
          const msg = "Password minimal 10 karakter sesuai standar keamanan.";
          setErrorMessage(msg);
          toast.warning(msg, "Validasi Password");
          setIsLoading(false);
          return;
        }
        await register({
          email,
          password,
          full_name: fullName,
          phone: phone || null,
          accept_terms: true,
        });
        toast.success("Selamat datang di KALENGO! Akun Anda berhasil dibuat.", "Registrasi Sukses");
      }
      onSuccess?.();
      onClose();
    } catch (err) {
      let msg = "Gagal memproses autentikasi. Silakan periksa kembali data Anda.";
      if (err instanceof ApiError) {
        msg = err.message;
      }
      setErrorMessage(msg);
      toast.error(msg, "Autentikasi Gagal");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div
      className="modal-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget && !isLoading) onClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="auth-modal-title"
    >
      <div className="modal-window" style={{ maxWidth: "460px" }}>
        <header className="modal-header" style={{ backgroundColor: "var(--lemon, #ffd23f)" }}>
          <h2 id="auth-modal-title" className="modal-title">
            {mode === "login" ? "Masuk ke KALENGO" : "Daftar Akun Baru"}
          </h2>
          <button
            type="button"
            className="btn btn--sm btn--secondary"
            onClick={onClose}
            disabled={isLoading}
            aria-label="Tutup form akun"
            style={{ padding: "0.35rem 0.65rem" }}
          >
            ✕
          </button>
        </header>

        <form onSubmit={handleSubmit} className="modal-body" style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          {errorMessage && (
            <div
              role="alert"
              style={{
                backgroundColor: "#ffe0dc",
                border: "2px solid var(--tomato, #ff4b3e)",
                borderRadius: "var(--radius-sm, 0.5rem)",
                padding: "0.75rem",
                color: "var(--tomato, #ff4b3e)",
                fontWeight: 800,
                fontSize: "0.875rem",
              }}
            >
              ⚠️ {errorMessage}
            </div>
          )}

          {mode === "register" && (
            <div>
              <label style={{ display: "block", fontSize: "0.8125rem", fontWeight: 800, marginBottom: "0.25rem" }}>
                Nama Lengkap *
              </label>
              <input
                type="text"
                required
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                placeholder="Nama Anda"
                style={{
                  width: "100%",
                  padding: "0.5rem 0.75rem",
                  border: "2px solid #1b1b2f",
                  borderRadius: "0.5rem",
                  fontWeight: 700,
                }}
              />
            </div>
          )}

          <div>
            <label style={{ display: "block", fontSize: "0.8125rem", fontWeight: 800, marginBottom: "0.25rem" }}>
              Alamat Email *
            </label>
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="nama@email.com"
              style={{
                width: "100%",
                padding: "0.5rem 0.75rem",
                border: "2px solid #1b1b2f",
                borderRadius: "0.5rem",
                fontWeight: 700,
              }}
            />
          </div>

          <div>
            <label style={{ display: "block", fontSize: "0.8125rem", fontWeight: 800, marginBottom: "0.25rem" }}>
              Password * {mode === "register" && "(Minimal 10 karakter)"}
            </label>
            <input
              type="password"
              required
              minLength={mode === "register" ? 10 : 1}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••••"
              style={{
                width: "100%",
                padding: "0.5rem 0.75rem",
                border: "2px solid #1b1b2f",
                borderRadius: "0.5rem",
                fontWeight: 700,
              }}
            />
          </div>

          {mode === "register" && (
            <div>
              <label style={{ display: "block", fontSize: "0.8125rem", fontWeight: 800, marginBottom: "0.25rem" }}>
                Nomor HP (Opsional)
              </label>
              <input
                type="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="08123456789"
                style={{
                  width: "100%",
                  padding: "0.5rem 0.75rem",
                  border: "2px solid #1b1b2f",
                  borderRadius: "0.5rem",
                  fontWeight: 700,
                }}
              />
            </div>
          )}

          {mode === "register" && (
            <label style={{ display: "flex", alignItems: "center", gap: "0.5rem", cursor: "pointer", fontSize: "0.8125rem", fontWeight: 700 }}>
              <input
                type="checkbox"
                required
                checked={acceptTerms}
                onChange={(e) => setAcceptTerms(e.target.checked)}
                style={{ width: "16px", height: "16px" }}
              />
              <span>Saya menyetujui Ketentuan Layanan & Kebijakan Privasi KALENGO.</span>
            </label>
          )}

          <button
            type="submit"
            className="btn btn--primary"
            disabled={isLoading}
            style={{ width: "100%", marginTop: "0.5rem" }}
          >
            {isLoading
              ? "Memproses..."
              : mode === "login"
                ? "Masuk Sekarang 👉"
                : "Daftar Akun Baru ✨"}
          </button>

          <div style={{ textAlign: "center", fontSize: "0.875rem", marginTop: "0.25rem" }}>
            {mode === "login" ? (
              <span>
                Belum punya akun?{" "}
                <button
                  type="button"
                  onClick={() => {
                    setMode("register");
                    setErrorMessage(null);
                  }}
                  style={{
                    background: "none",
                    border: "none",
                    fontWeight: 800,
                    color: "var(--ink, #1b1b2f)",
                    cursor: "pointer",
                    textDecoration: "underline",
                  }}
                >
                  Daftar di sini
                </button>
              </span>
            ) : (
              <span>
                Sudah punya akun?{" "}
                <button
                  type="button"
                  onClick={() => {
                    setMode("login");
                    setErrorMessage(null);
                  }}
                  style={{
                    background: "none",
                    border: "none",
                    fontWeight: 800,
                    color: "var(--ink, #1b1b2f)",
                    cursor: "pointer",
                    textDecoration: "underline",
                  }}
                >
                  Masuk di sini
                </button>
              </span>
            )}
          </div>

          <div
            style={{
              borderTop: "2px dashed #ddd",
              paddingTop: "0.75rem",
              textAlign: "center",
            }}
          >
            <button
              type="button"
              className="btn btn--secondary btn--sm"
              onClick={onClose}
              style={{ width: "100%" }}
            >
              Lanjut Belanja sebagai Tamu (Guest) 🛒
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
