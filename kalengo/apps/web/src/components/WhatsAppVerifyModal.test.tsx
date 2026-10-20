import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { WhatsAppVerifyModal } from "./WhatsAppVerifyModal";

const mockToast = {
  success: vi.fn(),
  error: vi.fn(),
  info: vi.fn(),
  remove: vi.fn(),
  toasts: [],
};

vi.mock("../context/useToast", () => ({
  useToast: () => mockToast,
}));

describe("WhatsAppVerifyModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("tidak merender apa pun saat isOpen bernilai false", () => {
    const { container } = render(
      <WhatsAppVerifyModal
        isOpen={false}
        phone="081234567890"
        onClose={vi.fn()}
        onVerified={vi.fn()}
      />,
    );
    expect(container.firstChild).toBeNull();
  });

  test("merender judul, nomor telepon, dan banner simulasi saat isOpen true", () => {
    render(
      <WhatsAppVerifyModal
        isOpen={true}
        phone="081234567890"
        onClose={vi.fn()}
        onVerified={vi.fn()}
      />,
    );

    expect(screen.getByText(/Verifikasi Nomor WhatsApp 💬/i)).toBeInTheDocument();
    expect(screen.getByText("081234567890")).toBeInTheDocument();
    expect(
      screen.getByText(/Kami telah mengirimkan 4-digit kode OTP ke nomor WhatsApp/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/\[Simulasi WhatsApp Gateway\]: Kode OTP Anda adalah/i),
    ).toBeInTheDocument();
  });

  test("tombol 'Gunakan Kode Ini' mengisi input OTP secara instan", () => {
    render(
      <WhatsAppVerifyModal
        isOpen={true}
        phone="081234567890"
        onClose={vi.fn()}
        onVerified={vi.fn()}
      />,
    );

    const input = screen.getByLabelText<HTMLInputElement>(/Masukkan 4-Digit Kode OTP/i);
    expect(input.value).toBe("");

    const fillButton = screen.getByRole("button", { name: /Gunakan Kode Ini/i });
    fireEvent.click(fillButton);

    expect(input.value.length).toBe(4);
    expect(/^\d{4}$/.test(input.value)).toBe(true);
  });

  test("menangani kode OTP yang salah dengan menampilkan error dan toast", () => {
    render(
      <WhatsAppVerifyModal
        isOpen={true}
        phone="081234567890"
        onClose={vi.fn()}
        onVerified={vi.fn()}
      />,
    );

    const input = screen.getByLabelText(/Masukkan 4-Digit Kode OTP/i);
    // Masukkan kode yang sengaja salah
    fireEvent.change(input, { target: { value: "0000" } });

    const submitBtn = screen.getByRole("button", { name: /Verifikasi Sekarang/i });
    fireEvent.click(submitBtn);

    expect(
      screen.getByText(/✕ Kode OTP salah. Silakan periksa kembali pesan WhatsApp Anda./i),
    ).toBeInTheDocument();
    expect(mockToast.error).toHaveBeenCalledWith(
      "✕ Kode OTP salah. Silakan periksa kembali pesan WhatsApp Anda.",
      "Verifikasi Gagal",
    );
  });

  test("memverifikasi kode yang cocok, memicu toast sukses dan callback onVerified", async () => {
    const onVerified = vi.fn();
    const onClose = vi.fn();

    render(
      <WhatsAppVerifyModal
        isOpen={true}
        phone="081234567890"
        onClose={onClose}
        onVerified={onVerified}
      />,
    );

    // Gunakan kode simulasi
    const fillButton = screen.getByRole("button", { name: /Gunakan Kode Ini/i });
    fireEvent.click(fillButton);

    const submitBtn = screen.getByRole("button", { name: /Verifikasi Sekarang/i });
    fireEvent.click(submitBtn);

    expect(mockToast.success).toHaveBeenCalledWith(
      "✓ Nomor WhatsApp 081234567890 berhasil diverifikasi! 🛡️",
      "Verifikasi Berhasil",
    );

    await waitFor(() => {
      expect(onVerified).toHaveBeenCalledWith("081234567890");
      expect(onClose).toHaveBeenCalled();
    });
  });

  test("tombol 'Batal / Ubah Nomor' menutup modal", () => {
    const onClose = vi.fn();
    render(
      <WhatsAppVerifyModal
        isOpen={true}
        phone="081234567890"
        onClose={onClose}
        onVerified={vi.fn()}
      />,
    );

    const cancelBtn = screen.getByRole("button", { name: /Batal \/ Ubah Nomor/i });
    fireEvent.click(cancelBtn);

    expect(onClose).toHaveBeenCalled();
  });

  test("countdown kirim ulang kode dan pengiriman kode baru saat 0s", () => {
    vi.useFakeTimers();

    render(
      <WhatsAppVerifyModal
        isOpen={true}
        phone="081234567890"
        onClose={vi.fn()}
        onVerified={vi.fn()}
      />,
    );

    // Saat awal countdown aktif 60s
    const resendBtn = screen.getByRole("button", { name: /Kirim Ulang Kode \(60s\)/i });
    expect(resendBtn).toBeDisabled();

    // Maju 60 detik
    act(() => {
      vi.advanceTimersByTime(60000);
    });

    const activeResendBtn = screen.getByRole("button", { name: /^Kirim Ulang Kode$/i });
    expect(activeResendBtn).not.toBeDisabled();

    fireEvent.click(activeResendBtn);

    expect(mockToast.info).toHaveBeenCalledWith(
      "Kode OTP WhatsApp baru telah dikirimkan!",
      "WhatsApp OTP",
    );

    vi.useRealTimers();
  });
});
