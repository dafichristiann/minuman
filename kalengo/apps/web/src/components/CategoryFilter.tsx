import React from "react";
import type { Category } from "../api/catalog";

export interface CategoryFilterProps {
  categories: readonly Category[];
  selectedCategory: string; // "all" atau slug/id
  onSelectCategory: (slugOrId: string) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
}

export const CategoryFilter: React.FC<CategoryFilterProps> = ({
  categories,
  selectedCategory,
  onSelectCategory,
  searchQuery,
  onSearchChange,
}) => {
  const defaultCategories: Array<{ id: string; slug: string; name: string }> = [
    { id: "all", slug: "all", name: "Semua Minuman" },
    { id: "soda", slug: "soda", name: "Soda" },
    { id: "kopi-susu", slug: "kopi-susu", name: "Kopi Susu" },
    { id: "teh", slug: "teh", name: "Teh" },
    { id: "rendah-gula", slug: "rendah-gula", name: "Rendah Gula" },
    { id: "buah", slug: "buah", name: "Rasa Buah" },
  ];

  // Gunakan kategori dari props jika tersedia, jika kosong gunakan defaultCategories
  const list = categories.length > 0
    ? [{ id: "all", slug: "all", name: "Semua Minuman" }, ...categories]
    : defaultCategories;

  return (
    <div className="catalog-controls" role="search" aria-label="Filter dan Cari Produk">
      <div className="search-bar-wrapper">
        <label htmlFor="product-search" className="visually-hidden">
          Cari minuman kaleng
        </label>
        <span className="search-bar__icon" aria-hidden="true">🔍</span>
        <input
          id="product-search"
          type="search"
          className="search-bar__input"
          placeholder="Cari rasa, soda, kopi, atau teh favoritmu..."
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          autoComplete="off"
        />
        {searchQuery.length > 0 && (
          <button
            type="button"
            className="search-bar__clear"
            onClick={() => onSearchChange("")}
            aria-label="Bersihkan pencarian"
          >
            ✕
          </button>
        )}
      </div>

      <div className="filter-group" role="group" aria-label="Kategori Produk">
        {list.map((cat) => {
          const isSelected = selectedCategory === cat.slug || selectedCategory === cat.id;
          return (
            <button
              key={cat.id || cat.slug}
              type="button"
              aria-pressed={isSelected}
              className={`filter-pill ${isSelected ? "is-active" : ""}`}
              onClick={() => onSelectCategory(cat.slug || cat.id)}
            >
              {cat.name}
            </button>
          );
        })}
      </div>
    </div>
  );
};
