import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { SearchableSelect } from "./SearchableSelect";

describe("SearchableSelect component", () => {
  const options = [
    { value: "DKI Jakarta", label: "DKI Jakarta" },
    { value: "Jawa Barat", label: "Jawa Barat" },
    { value: "Jawa Tengah", label: "Jawa Tengah" },
  ];

  test("renders label, placeholder, and initial value", () => {
    render(
      <SearchableSelect
        id="test-prov"
        label="Provinsi"
        placeholder="Pilih Provinsi"
        value="Jawa Barat"
        options={options}
        onChange={vi.fn()}
      />,
    );

    expect(screen.getByLabelText(/provinsi/i)).toBeInTheDocument();
    expect(screen.getByDisplayValue("Jawa Barat")).toBeInTheDocument();
  });

  test("filters options case-insensitively as user types", () => {
    render(
      <SearchableSelect
        id="test-prov"
        label="Provinsi"
        placeholder="Pilih Provinsi"
        value=""
        options={options}
        onChange={vi.fn()}
      />,
    );

    const input = screen.getByRole("combobox");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "jawa b" } });

    expect(screen.getByRole("option", { name: /jawa barat/i })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /dki jakarta/i })).not.toBeInTheDocument();
  });

  test("selects option on click and calls onChange", () => {
    const handleChange = vi.fn();
    render(
      <SearchableSelect
        id="test-prov"
        label="Provinsi"
        placeholder="Pilih Provinsi"
        value=""
        options={options}
        onChange={handleChange}
      />,
    );

    const input = screen.getByRole("combobox");
    fireEvent.focus(input);

    const option = screen.getByRole("option", { name: /dki jakarta/i });
    fireEvent.click(option);

    expect(handleChange).toHaveBeenCalledWith(
      "DKI Jakarta",
      expect.objectContaining({ value: "DKI Jakarta", label: "DKI Jakarta" }),
    );
  });

  test("keyboard navigation: ArrowDown, ArrowUp, Enter, and Escape", () => {
    const handleChange = vi.fn();
    render(
      <SearchableSelect
        id="test-prov"
        label="Provinsi"
        placeholder="Pilih Provinsi"
        value=""
        options={options}
        onChange={handleChange}
      />,
    );

    const input = screen.getByRole("combobox");
    fireEvent.focus(input);

    // Arrow down moves to first item
    fireEvent.keyDown(input, { key: "ArrowDown" });
    // Arrow down moves to second item
    fireEvent.keyDown(input, { key: "ArrowDown" });
    // Enter selects active item ("Jawa Barat")
    fireEvent.keyDown(input, { key: "Enter" });

    expect(handleChange).toHaveBeenCalledWith("Jawa Barat", expect.anything());

    // Press escape to close
    fireEvent.focus(input);
    fireEvent.keyDown(input, { key: "Escape" });
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("clear button clears value and calls onChange with empty string", () => {
    const handleChange = vi.fn();
    render(
      <SearchableSelect
        id="test-prov"
        label="Provinsi"
        placeholder="Pilih Provinsi"
        value="Jawa Barat"
        options={options}
        onChange={handleChange}
      />,
    );

    const clearBtn = screen.getByRole("button", { name: /hapus pilihan/i });
    fireEvent.click(clearBtn);

    expect(handleChange).toHaveBeenCalledWith("", undefined);
  });

  test("displays empty state and supports allowCustomInput", () => {
    const handleChange = vi.fn();
    render(
      <SearchableSelect
        id="test-prov"
        label="Provinsi"
        placeholder="Pilih Provinsi"
        value=""
        options={options}
        onChange={handleChange}
        allowCustomInput={true}
      />,
    );

    const input = screen.getByRole("combobox");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "Nusantara" } });

    expect(screen.getByText(/tidak ditemukan hasil untuk "nusantara"/i)).toBeInTheDocument();

    const customBtn = screen.getByRole("button", { name: /gunakan "nusantara"/i });
    fireEvent.click(customBtn);

    expect(handleChange).toHaveBeenCalledWith("Nusantara");
  });

  test("renders valid and invalid feedback badges when touched", () => {
    const { rerender } = render(
      <SearchableSelect
        id="test-prov"
        label="Provinsi"
        placeholder="Pilih Provinsi"
        value=""
        options={options}
        onChange={vi.fn()}
        isTouched={true}
        isValid={false}
        feedbackText="Pilih provinsi yang sesuai"
      />,
    );

    expect(screen.getByText(/✕ Pilih provinsi yang sesuai/i)).toBeInTheDocument();

    rerender(
      <SearchableSelect
        id="test-prov"
        label="Provinsi"
        placeholder="Pilih Provinsi"
        value="DKI Jakarta"
        options={options}
        onChange={vi.fn()}
        isTouched={true}
        isValid={true}
      />,
    );

    expect(screen.getByText(/✓ Provinsi Terpilih/i)).toBeInTheDocument();
  });
});
