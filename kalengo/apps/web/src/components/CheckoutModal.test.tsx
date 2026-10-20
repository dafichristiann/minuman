import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { CheckoutModal } from "./CheckoutModal";

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

vi.mock("../context/AuthContext", () => ({
  useAuth: () => ({
    user: null,
    isAuthenticated: false,
    token: null,
  }),
}));

vi.mock("../context/CartContext", () => ({
  useCart: () => ({
    subtotal: 50000,
    items: [
      {
        variant_id: "var-1",
        product_name: "Kalengo Spark Lemon",
        sku: "KLG-SPK-LEM",
        pack_size: "Can 250ml",
        quantity: 2,
        unit_price: 25000,
        line_total: 50000,
      },
    ],
    clearCart: vi.fn(),
  }),
}));

describe("CheckoutModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("validasi real-time nama penerima (min 2 karakter)", () => {
    render(
      <CheckoutModal
        isOpen={true}
        onClose={vi.fn()}
        onOrderCreated={vi.fn()}
        onRequireAuth={vi.fn()}
      />,
    );

    const recipientInput = screen.getByLabelText(/nama penerima/i);

    // Saat touched tapi hanya 1 karakter
    fireEvent.change(recipientInput, { target: { value: "A" } });
    expect(
      screen.getByText(/✕ Minimal 2 karakter alfabet/i),
    ).toBeInTheDocument();
    expect(recipientInput).toHaveClass("input--invalid");

    // Saat valid (>= 2 karakter alfabet)
    fireEvent.change(recipientInput, { target: { value: "Budi Santoso" } });
    expect(screen.getByText(/✓ Nama Valid/i)).toBeInTheDocument();
    expect(recipientInput).toHaveClass("input--valid");
  });

  test("validasi nomor HP format Indonesia dan indikator status verifikasi", async () => {
    render(
      <CheckoutModal
        isOpen={true}
        onClose={vi.fn()}
        onOrderCreated={vi.fn()}
        onRequireAuth={vi.fn()}
      />,
    );

    const phoneInput = screen.getByLabelText(/nomor hp/i);

    // Nomor salah format
    fireEvent.change(phoneInput, { target: { value: "12345" } });
    expect(
      screen.getByText(/✕ Format tidak valid/i),
    ).toBeInTheDocument();
    expect(phoneInput).toHaveClass("input--invalid");

    // Nomor valid Indonesia -> Belum Terverifikasi & muncul tombol verifikasi
    fireEvent.change(phoneInput, { target: { value: "081234567890" } });
    expect(screen.getByText(/⚠️ Belum Terverifikasi/i)).toBeInTheDocument();
    const verifyWaBtn = screen.getByRole("button", { name: /💬 Verifikasi WhatsApp/i });
    expect(verifyWaBtn).toBeInTheDocument();
    expect(phoneInput).toHaveClass("input--valid");

    // Klik verifikasi -> buka WhatsAppVerifyModal
    fireEvent.click(verifyWaBtn);
    expect(screen.getByText(/Verifikasi Nomor WhatsApp 💬/i)).toBeInTheDocument();

    // Verifikasi dengan kode simulasi
    fireEvent.click(screen.getByRole("button", { name: /Gunakan Kode Ini/i }));
    fireEvent.click(screen.getByRole("button", { name: /Verifikasi Sekarang/i }));

    // Setelah terverifikasi -> badge terverifikasi muncul
    await waitFor(() => {
      expect(screen.getByText(/✓ Terverifikasi WhatsApp 🛡️/i)).toBeInTheDocument();
    });

    // Jika nomor diubah, status verifikasi kembali belum terverifikasi
    fireEvent.change(phoneInput, { target: { value: "081234567899" } });
    expect(screen.getByText(/⚠️ Belum Terverifikasi/i)).toBeInTheDocument();
  });

  test("validasi alamat jalan dan kode pos", () => {
    render(
      <CheckoutModal
        isOpen={true}
        onClose={vi.fn()}
        onOrderCreated={vi.fn()}
        onRequireAuth={vi.fn()}
      />,
    );

    const streetInput = screen.getByLabelText(/alamat jalan & patokan/i);
    const postalInput = screen.getByLabelText(/kode pos/i);

    // Alamat terlalu pendek (< 10 huruf / karakter)
    fireEvent.change(streetInput, { target: { value: "Jl. Mawar" } });
    expect(
      screen.getByText(/✕ Alamat minimal 10 huruf beserta nomor\/patokan/i),
    ).toBeInTheDocument();

    // Alamat lengkap
    fireEvent.change(streetInput, {
      target: { value: "Jl. Mawar Melati No. 10 Jakarta" },
    });
    expect(screen.getByText(/✓ Alamat Lengkap/i)).toBeInTheDocument();

    // Kode pos tidak 5 digit angka
    fireEvent.change(postalInput, { target: { value: "123" } });
    expect(
      screen.getByText(/✕ Kode pos wajib 5 digit angka/i),
    ).toBeInTheDocument();

    // Kode pos valid
    fireEvent.change(postalInput, { target: { value: "12345" } });
    expect(screen.getByText(/✓ 5 Digit Valid/i)).toBeInTheDocument();
  });

  test("submit saat form tidak valid memicu shake dan toast error", () => {
    render(
      <CheckoutModal
        isOpen={true}
        onClose={vi.fn()}
        onOrderCreated={vi.fn()}
        onRequireAuth={vi.fn()}
      />,
    );

    const submitBtn = screen.getByRole("button", { name: /bayar sekarang/i });
    fireEvent.click(submitBtn);

    expect(mockToast.error).toHaveBeenCalledWith(
      "Mohon perbaiki isian yang belum sesuai sebelum melanjutkan.",
      "Validasi Gagal",
    );
  });

  test("menampilkan badge ✓ Data Lengkap dan memproses pesanan setelah verifikasi WhatsApp", async () => {
    const onClose = vi.fn();
    const onOrderCreated = vi.fn();

    render(
      <CheckoutModal
        isOpen={true}
        onClose={onClose}
        onOrderCreated={onOrderCreated}
        onRequireAuth={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText(/nama penerima/i), {
      target: { value: "Siti Rahma" },
    });
    fireEvent.change(screen.getByLabelText(/nomor hp/i), {
      target: { value: "081987654321" },
    });
    fireEvent.change(screen.getByLabelText(/alamat jalan & patokan/i), {
      target: { value: "Jl. Kebon Jeruk Barat No. 88, RT 02/05" },
    });
    fireEvent.change(screen.getByLabelText(/provinsi/i), {
      target: { value: "DKI Jakarta" },
    });
    fireEvent.change(screen.getByLabelText(/kota \/ kabupaten/i), {
      target: { value: "Jakarta Barat" },
    });
    fireEvent.change(screen.getByLabelText(/kelurahan \/ kecamatan/i), {
      target: { value: "Kebon Jeruk" },
    });
    fireEvent.change(screen.getByLabelText(/kode pos/i), {
      target: { value: "11530" },
    });

    // Coba submit sebelum nomor WhatsApp diverifikasi
    const submitBtn = screen.getByRole("button", { name: /bayar sekarang/i });
    fireEvent.click(submitBtn);

    // Menampilkan toast peringatan dan otomatis membuka WhatsAppVerifyModal
    expect(mockToast.error).toHaveBeenCalledWith(
      "Silakan verifikasi nomor WhatsApp Anda terlebih dahulu.",
      "Verifikasi WhatsApp",
    );
    expect(screen.getByText(/Verifikasi Nomor WhatsApp 💬/i)).toBeInTheDocument();

    // Verifikasi WhatsApp
    fireEvent.click(screen.getByRole("button", { name: /Gunakan Kode Ini/i }));
    fireEvent.click(screen.getByRole("button", { name: /Verifikasi Sekarang/i }));

    // Setelah modal tertutup dan terverifikasi, badge data lengkap muncul
    await waitFor(() => {
      expect(screen.getByText("✓ Data Lengkap")).toBeInTheDocument();
      expect(screen.getByText(/✓ Terverifikasi WhatsApp 🛡️/i)).toBeInTheDocument();
    });

    // Klik Bayar Sekarang setelah data lengkap & terverifikasi
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(onClose).toHaveBeenCalled();
      expect(onOrderCreated).toHaveBeenCalledWith(
        expect.objectContaining({
          status: "pending_payment",
          subtotal: 50000,
          shipping_cost: 15000,
          total: 65000,
        }),
      );
      expect(mockToast.success).toHaveBeenCalledWith(
        "Pesanan berhasil dibuat! Silakan selesaikan pembayaran.",
        "Pesanan Dibuat",
      );
    });
  });

  test("memproses checkout tamu secara andal dengan nomor +62 dan verifikasi WhatsApp", async () => {
    const onClose = vi.fn();
    const onOrderCreated = vi.fn();

    render(
      <CheckoutModal
        isOpen={true}
        onClose={onClose}
        onOrderCreated={onOrderCreated}
        onRequireAuth={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText(/nama penerima/i), {
      target: { value: "Ahmad Dahlan" },
    });
    fireEvent.change(screen.getByLabelText(/nomor hp/i), {
      target: { value: "+628123456789" },
    });
    fireEvent.change(screen.getByLabelText(/alamat jalan & patokan/i), {
      target: { value: "Jl. Sudirman No. 1" },
    });
    fireEvent.change(screen.getByLabelText(/provinsi/i), {
      target: { value: "DKI Jakarta" },
    });
    fireEvent.change(screen.getByLabelText(/kota \/ kabupaten/i), {
      target: { value: "Jakarta Pusat" },
    });
    fireEvent.change(screen.getByLabelText(/kelurahan \/ kecamatan/i), {
      target: { value: "Gambir" },
    });
    fireEvent.change(screen.getByLabelText(/kode pos/i), {
      target: { value: "10110" },
    });

    // Buka verifikasi WA lewat tombol
    const verifyWaBtn = screen.getByRole("button", { name: /💬 Verifikasi WhatsApp/i });
    fireEvent.click(verifyWaBtn);

    // Verifikasi
    fireEvent.click(screen.getByRole("button", { name: /Gunakan Kode Ini/i }));
    fireEvent.click(screen.getByRole("button", { name: /Verifikasi Sekarang/i }));

    await waitFor(() => {
      expect(screen.getByText(/✓ Terverifikasi WhatsApp 🛡️/i)).toBeInTheDocument();
    });

    const submitBtn = screen.getByRole("button", { name: /bayar sekarang/i });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(onClose).toHaveBeenCalled();
      expect(onOrderCreated).toHaveBeenCalled();
    });
  });

  test("memilih kecamatan dari daftar otomatis mengisi kode pos", () => {
    render(
      <CheckoutModal
        isOpen={true}
        onClose={vi.fn()}
        onOrderCreated={vi.fn()}
        onRequireAuth={vi.fn()}
      />,
    );

    const provInput = screen.getByLabelText(/provinsi/i);
    fireEvent.focus(provInput);
    fireEvent.change(provInput, { target: { value: "Jawa Barat" } });
    const jabarOpt = screen.getByRole("option", { name: /jawa barat/i });
    fireEvent.click(jabarOpt);

    const cityInput = screen.getByLabelText(/kota \/ kabupaten/i);
    fireEvent.focus(cityInput);
    fireEvent.change(cityInput, { target: { value: "Kota Bandung" } });
    const bdgOpt = screen.getByRole("option", { name: /kota bandung/i });
    fireEvent.click(bdgOpt);

    const districtInput = screen.getByLabelText(/kelurahan \/ kecamatan/i);
    fireEvent.focus(districtInput);
    fireEvent.change(districtInput, { target: { value: "Coblong" } });
    const distOpt = screen.getByRole("option", { name: /coblong/i });
    fireEvent.click(distOpt);

    const postalInput = screen.getByLabelText<HTMLInputElement>(/kode pos/i);
    expect(postalInput.value).toBe("40132");
  });
});
