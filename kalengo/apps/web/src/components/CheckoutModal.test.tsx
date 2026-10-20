import { fireEvent, render, screen } from "@testing-library/react";
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

  test("validasi nomor HP format Indonesia", () => {
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

    // Nomor valid Indonesia
    fireEvent.change(phoneInput, { target: { value: "081234567890" } });
    expect(screen.getByText(/✓ Nomor HP Valid/i)).toBeInTheDocument();
    expect(phoneInput).toHaveClass("input--valid");
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

    // Alamat terlalu pendek (< 10 karakter)
    fireEvent.change(streetInput, { target: { value: "Jl. Mawar" } });
    expect(
      screen.getByText(/✕ Alamat minimal 10 karakter/i),
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

    const submitBtn = screen.getByRole("button", { name: /lanjut ke konfirmasi/i });
    fireEvent.click(submitBtn);

    expect(mockToast.error).toHaveBeenCalledWith(
      "Mohon perbaiki isian yang belum sesuai sebelum melanjutkan.",
      "Validasi Gagal",
    );
  });

  test("menampilkan badge ✓ Data Lengkap dan membuka ConfirmModal saat valid", () => {
    render(
      <CheckoutModal
        isOpen={true}
        onClose={vi.fn()}
        onOrderCreated={vi.fn()}
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
    fireEvent.change(screen.getByLabelText(/kota \/ kabupaten/i), {
      target: { value: "Jakarta Barat" },
    });
    fireEvent.change(screen.getByLabelText(/provinsi/i), {
      target: { value: "DKI Jakarta" },
    });
    fireEvent.change(screen.getByLabelText(/kode pos/i), {
      target: { value: "11530" },
    });

    // Badge data lengkap muncul
    expect(screen.getByText("✓ Data Lengkap")).toBeInTheDocument();

    // Klik lanjut ke konfirmasi
    const submitBtn = screen.getByRole("button", { name: /lanjut ke konfirmasi/i });
    fireEvent.click(submitBtn);

    // Modal konfirmasi terbuka dengan ringkasan
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(screen.getByText(/Konfirmasi Pesanan/i)).toBeInTheDocument();
    expect(screen.getByText(/Siti Rahma/i)).toBeInTheDocument();
    expect(screen.getByText(/081987654321/i)).toBeInTheDocument();
  });
});
