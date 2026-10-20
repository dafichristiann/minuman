import { describe, expect, test } from "vitest";
import {
  findPostalCode,
  getCitiesByProvince,
  getDistrictsByCity,
  getProvinces,
  INDONESIA_REGIONS,
} from "./indonesiaRegions";

describe("indonesiaRegions dataset & helpers", () => {
  test("getProvinces returns expected list of key provinces", () => {
    const provinces = getProvinces();
    expect(provinces.length).toBeGreaterThanOrEqual(13);
    expect(provinces).toContain("DKI Jakarta");
    expect(provinces).toContain("Jawa Barat");
    expect(provinces).toContain("Banten");
    expect(provinces).toContain("Jawa Tengah");
    expect(provinces).toContain("DI Yogyakarta");
    expect(provinces).toContain("Jawa Timur");
    expect(provinces).toContain("Bali");
    expect(provinces).toContain("Sumatera Utara");
    expect(provinces).toContain("Sulawesi Selatan");
  });

  test("getCitiesByProvince returns cities for Jawa Barat", () => {
    const jabarCities = getCitiesByProvince("Jawa Barat");
    expect(jabarCities).toContain("Kota Bandung");
    expect(jabarCities).toContain("Kota Bogor");
    expect(jabarCities).toContain("Kota Depok");
    expect(jabarCities).toContain("Kota Bekasi");
  });

  test("getCitiesByProvince is case insensitive and handles unknown gracefully", () => {
    expect(getCitiesByProvince("dki jakarta")).toContain("Kota Jakarta Selatan");
    expect(getCitiesByProvince("")).toEqual([]);
    expect(getCitiesByProvince("Atlantis")).toEqual([]);
  });

  test("getDistrictsByCity returns districts with postal codes", () => {
    const districts = getDistrictsByCity("Jawa Barat", "Kota Bandung");
    expect(districts.length).toBeGreaterThan(0);
    const coblong = districts.find((d) => d.name === "Coblong");
    expect(coblong).toBeDefined();
    expect(coblong?.postalCode).toBe("40132");
  });

  test("findPostalCode resolves exact and partial matches", () => {
    expect(findPostalCode("Jawa Barat", "Kota Bandung", "Coblong")).toBe("40132");
    expect(findPostalCode("DKI Jakarta", "Kota Jakarta Selatan", "Kebayoran Baru")).toBe("12110");
    expect(findPostalCode("Jawa Barat", "Kota Bandung", "Tidak Ada")).toBeNull();
    expect(findPostalCode("", "", "")).toBeNull();
  });

  test("all districts across all regions have valid 5-digit postal codes", () => {
    const fiveDigitRegex = /^[0-9]{5}$/;
    for (const prov of INDONESIA_REGIONS) {
      for (const city of prov.cities) {
        expect(city.districts.length).toBeGreaterThan(0);
        for (const dist of city.districts) {
          expect(dist.postalCode).toMatch(fiveDigitRegex);
        }
      }
    }
  });
});
