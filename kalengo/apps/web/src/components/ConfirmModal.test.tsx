import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { ConfirmModal } from "./ConfirmModal";

describe("ConfirmModal", () => {
  test("tidak merender apa-apa saat isOpen false", () => {
    const { container } = render(
      <ConfirmModal
        isOpen={false}
        title="Konfirmasi"
        message="Apakah yakin?"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    expect(container.firstChild).toBeNull();
  });

  test("merender judul, pesan, dan tombol saat isOpen true", () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();

    render(
      <ConfirmModal
        isOpen={true}
        title="Hapus Minuman?"
        message="Hapus minuman ini dari keranjang?"
        confirmText="Hapus Sekarang"
        cancelText="Batalkan"
        onConfirm={onConfirm}
        onCancel={onCancel}
      />,
    );

    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(screen.getByText("Hapus Minuman?")).toBeInTheDocument();
    expect(screen.getByText("Hapus minuman ini dari keranjang?")).toBeInTheDocument();

    const confirmBtn = screen.getByRole("button", { name: "Hapus Sekarang" });
    fireEvent.click(confirmBtn);
    expect(onConfirm).toHaveBeenCalledTimes(1);

    const cancelBtn = screen.getByRole("button", { name: "Batalkan" });
    fireEvent.click(cancelBtn);
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  test("memanggil onCancel saat tombol Escape ditekan", () => {
    const onCancel = vi.fn();

    render(
      <ConfirmModal
        isOpen={true}
        title="Keluar Akun?"
        message="Yakin ingin keluar akun?"
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />,
    );

    fireEvent.keyDown(window, { key: "Escape" });
    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
