import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import { ToastProvider, useToast } from "./ToastContext";

function TestComponent() {
  const toast = useToast();

  return (
    <div>
      <button
        type="button"
        onClick={() => toast.success("Minuman berhasil ditambahkan!", "Sukses")}
      >
        Trigger Success
      </button>
      <button
        type="button"
        onClick={() => toast.error("Gagal menghubungi server.", "Error")}
      >
        Trigger Error
      </button>
      <button
        type="button"
        onClick={() => toast.warning("Stok hampir habis!", "Peringatan")}
      >
        Trigger Warning
      </button>
      <button
        type="button"
        onClick={() => toast.info("Pesanan sedang diproses.", "Informasi")}
      >
        Trigger Info
      </button>
    </div>
  );
}

describe("ToastContext & useToast", () => {
  test("menampilkan toast sukses saat dipanggil", () => {
    render(
      <ToastProvider>
        <TestComponent />
      </ToastProvider>,
    );

    act(() => {
      fireEvent.click(screen.getByRole("button", { name: "Trigger Success" }));
    });

    expect(screen.getByText("Minuman berhasil ditambahkan!")).toBeInTheDocument();
    expect(screen.getByText("Sukses")).toBeInTheDocument();
    expect(screen.getByText("SUKSES")).toBeInTheDocument();
  });

  test("menampilkan toast error dengan role alert", () => {
    render(
      <ToastProvider>
        <TestComponent />
      </ToastProvider>,
    );

    act(() => {
      fireEvent.click(screen.getByRole("button", { name: "Trigger Error" }));
    });

    const alert = screen.getByRole("alert");
    expect(alert).toHaveTextContent("Gagal menghubungi server.");
    expect(screen.getByText("GAGAL")).toBeInTheDocument();
  });

  test("dapat menutup toast secara manual melalui tombol dismiss", () => {
    render(
      <ToastProvider>
        <TestComponent />
      </ToastProvider>,
    );

    act(() => {
      fireEvent.click(screen.getByRole("button", { name: "Trigger Warning" }));
    });

    expect(screen.getByText("Stok hampir habis!")).toBeInTheDocument();

    const closeBtn = screen.getByRole("button", { name: "Tutup notifikasi" });
    act(() => {
      fireEvent.click(closeBtn);
    });

    expect(screen.queryByText("Stok hampir habis!")).not.toBeInTheDocument();
  });
});
