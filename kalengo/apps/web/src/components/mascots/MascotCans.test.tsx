import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  CanSodaLemon,
  CanKopiSusu,
  CanTehMelati,
  CanLiteZero,
  CanBuahMangga,
  CanSodaJahe,
  MascotCan,
  MASCOT_REGISTRY,
} from "./MascotCans";

describe("Mascot Cans", () => {
  it("renders all 6 distinct mascot can components with accessible roles", () => {
    const list = [
      { Component: CanSodaLemon, label: "Maskot Kalengo Soda Lemon" },
      { Component: CanKopiSusu, label: "Maskot Kalengo Kopi Susu" },
      { Component: CanTehMelati, label: "Maskot Kalengo Teh Melati" },
      { Component: CanLiteZero, label: "Maskot Kalengo Lite Zero" },
      { Component: CanBuahMangga, label: "Maskot Kalengo Rasa Mangga" },
      { Component: CanSodaJahe, label: "Maskot Kalengo Soda Jahe" },
    ];

    list.forEach(({ Component, label }) => {
      const { unmount } = render(<Component />);
      const el = screen.getByRole("img", { name: label });
      expect(el).toBeInTheDocument();
      expect(el.getAttribute("viewBox")).toBe("0 0 180 220");
      unmount();
    });
  });

  it("renders mascot through MascotCan dispatcher", () => {
    render(<MascotCan name="can-lite-zero" />);
    expect(screen.getByRole("img", { name: "Maskot Kalengo Lite Zero" })).toBeInTheDocument();
  });

  it("falls back gracefully to CanSodaLemon if mascot name is unknown", () => {
    render(<MascotCan name="can-unknown-flavor" />);
    expect(screen.getByRole("img", { name: "Maskot Kalengo Soda Lemon" })).toBeInTheDocument();
  });

  it("contains registry of all 6 mascots", () => {
    expect(Object.keys(MASCOT_REGISTRY)).toHaveLength(6);
    expect(MASCOT_REGISTRY["can-soda-lemon"]).toBeDefined();
    expect(MASCOT_REGISTRY["can-kopi-susu"]).toBeDefined();
    expect(MASCOT_REGISTRY["can-teh-melati"]).toBeDefined();
    expect(MASCOT_REGISTRY["can-lite-zero"]).toBeDefined();
    expect(MASCOT_REGISTRY["can-buah-mangga"]).toBeDefined();
    expect(MASCOT_REGISTRY["can-soda-jahe"]).toBeDefined();
  });
});
