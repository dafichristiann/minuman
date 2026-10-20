import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Order } from "../api/checkout";
import { PaymentModal } from "./PaymentModal";

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

const sampleOrder: Order = {
  id: "ord-test-123",
  order_no: "KLG-20261020-8888",
  status: "pending_payment",
  subtotal: 50000,
  discount: 5000,
  shipping_cost: 15000,
  total: 60000,
  courier: "jne",
  courier_service: "REG",
  expires_at: new Date(Date.now() + 15 * 60 * 1000).toISOString(),
  created_at: new Date().toISOString(),
  items: [
    {
      variant_id: "var-1",
      product_name: "Kalengo Spark Lemon",
      sku: "KLG-SPK-LEM",
      pack_size: 250,
      quantity: 2,
      unit_price: 25000,
      line_total: 50000,
    },
  ],
};

describe("PaymentModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("merender QRIS code, hitung mundur, dan panduan scan", () => {
    render(
      <PaymentModal
        isOpen={true}
        order={sampleOrder}
        onClose={vi.fn()}
        onPaymentSuccess={vi.fn()}
      />,
    );

    expect(screen.getByText(/Bayar Pesanan/i)).toBeInTheDocument();
    expect(screen.getByText(/KLG-20261020-8888/i)).toBeInTheDocument();
    expect(screen.getByText(/Sisa Waktu/i)).toBeInTheDocument();
    expect(screen.getByText(/QRIS • STANDAR NASIONAL/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/QRIS Code KALENGO/i)).toBeInTheDocument();
  });

  test("berpindah ke BCA Virtual Account dan menyalin nomor VA", async () => {
    const clipboardWriteText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: {
        writeText: clipboardWriteText,
      },
    });

    render(
      <PaymentModal
        isOpen={true}
        order={sampleOrder}
        onClose={vi.fn()}
        onPaymentSuccess={vi.fn()}
      />,
    );

    // Klik BCA Virtual Account
    const bcaBtn = screen.getByRole("button", { name: /BCA Virtual Account/i });
    fireEvent.click(bcaBtn);

    // Periksa nomor VA tertera
    expect(screen.getByText("8271 0812 9923 1184")).toBeInTheDocument();

    // Klik tombol Salin No. VA
    const copyBtn = screen.getByRole("button", {
      name: /Salin Nomor Virtual Account/i,
    });
    fireEvent.click(copyBtn);

    await waitFor(() => {
      expect(clipboardWriteText).toHaveBeenCalledWith("8271081299231184");
      expect(mockToast.success).toHaveBeenCalledWith(
        "✓ Nomor Virtual Account berhasil disalin!",
        "Berhasil Disalin",
      );
    });
  });

  test("berpindah ke kanal GoPay menampilkan simulasi notifikasi ponsel", () => {
    render(
      <PaymentModal
        isOpen={true}
        order={sampleOrder}
        onClose={vi.fn()}
        onPaymentSuccess={vi.fn()}
      />,
    );

    const gopayBtn = screen.getByRole("button", { name: /GoPay \/ E-Wallet/i });
    fireEvent.click(gopayBtn);

    expect(
      screen.getByText(/Notifikasi Terkirim ke Ponsel Anda/i),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Buka Aplikasi Gojek/i }),
    ).toBeInTheDocument();
  });

  test("mengklik Konfirmasi Bayar menjalankan simulasi verifikasi dan menampilkan struk lunas", async () => {
    const onPaymentSuccess = vi.fn();

    render(
      <PaymentModal
        isOpen={true}
        order={sampleOrder}
        onClose={vi.fn()}
        onPaymentSuccess={onPaymentSuccess}
      />,
    );

    const confirmPayBtn = screen.getByRole("button", {
      name: /Konfirmasi Bayar 👉/i,
    });
    fireEvent.click(confirmPayBtn);

    // Status memverifikasi dengan Midtrans
    expect(
      screen.getByText(/Memverifikasi transaksi dengan Midtrans/i),
    ).toBeInTheDocument();

    // Tunggu simulasi selesai
    await waitFor(
      () => {
        expect(screen.getByText(/Pembayaran Berhasil!/i)).toBeInTheDocument();
      },
      { timeout: 2500 },
    );

    expect(screen.getByText(/LUNAS \(SETTLEMENT\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Unduh Bukti Transaksi 📄/i)).toBeInTheDocument();
    expect(onPaymentSuccess).toHaveBeenCalled();
  });
});
