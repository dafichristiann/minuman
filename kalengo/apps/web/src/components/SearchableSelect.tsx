import React, { useEffect, useId, useMemo, useRef, useState } from "react";

export interface SearchableOption {
  readonly value: string;
  readonly label: string;
  readonly badge?: string | undefined;
}

export interface SearchableSelectProps {
  id: string;
  label: string;
  placeholder: string;
  value: string;
  options: readonly SearchableOption[];
  onChange: (value: string, item?: SearchableOption) => void;
  disabled?: boolean | undefined;
  isValid?: boolean | undefined;
  isTouched?: boolean | undefined;
  feedbackText?: string | undefined;
  required?: boolean | undefined;
  allowCustomInput?: boolean | undefined;
  className?: string | undefined;
}

export const SearchableSelect: React.FC<SearchableSelectProps> = ({
  id,
  label,
  placeholder,
  value,
  options,
  onChange,
  disabled = false,
  isValid,
  isTouched = false,
  feedbackText,
  required = false,
  allowCustomInput = false,
  className = "",
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const listboxRef = useRef<HTMLUListElement>(null);

  const [isOpen, setIsOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState<number>(-1);
  const listboxId = useId();

  // Sync internal search query with external value when closed
  useEffect(() => {
    if (!isOpen) {
      const v = value.trim().toLowerCase();
      const match = options.find(
        (opt) =>
          opt.value.toLowerCase() === v ||
          opt.label.toLowerCase() === v ||
          (v.length >= 3 && opt.value.toLowerCase().includes(v)),
      );
      setQuery(match ? match.label : value);
    }
  }, [value, options, isOpen]);

  // Filter options based on query
  const filteredOptions = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return options;
    return options.filter(
      (opt) =>
        opt.label.toLowerCase().includes(q) ||
        opt.value.toLowerCase().includes(q) ||
        (opt.badge && opt.badge.toLowerCase().includes(q)),
    );
  }, [options, query]);

  // Click outside listener
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setActiveIndex(-1);
        // Reset query back to value if custom input not allowed or empty
        if (!allowCustomInput) {
          const match = options.find((opt) => opt.value === value || opt.label === value);
          setQuery(match ? match.label : value);
        }
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [value, options, allowCustomInput]);

  const handleSelectOption = (opt: SearchableOption) => {
    onChange(opt.value, opt);
    setQuery(opt.label);
    setIsOpen(false);
    setActiveIndex(-1);
    inputRef.current?.focus();
  };

  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation();
    onChange("", undefined);
    setQuery("");
    setActiveIndex(-1);
    setIsOpen(false);
    inputRef.current?.focus();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (disabled) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (!isOpen) {
        setIsOpen(true);
        setActiveIndex(0);
      } else {
        setActiveIndex((prev) => (prev < filteredOptions.length - 1 ? prev + 1 : 0));
      }
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      if (!isOpen) {
        setIsOpen(true);
        setActiveIndex(filteredOptions.length - 1);
      } else {
        setActiveIndex((prev) => (prev > 0 ? prev - 1 : filteredOptions.length - 1));
      }
    } else if (e.key === "Enter") {
      if (isOpen && activeIndex >= 0 && activeIndex < filteredOptions.length) {
        e.preventDefault();
        const selected = filteredOptions[activeIndex];
        if (selected) {
          handleSelectOption(selected);
        }
      } else if (isOpen && allowCustomInput && query.trim()) {
        e.preventDefault();
        onChange(query.trim());
        setIsOpen(false);
      }
    } else if (e.key === "Escape") {
      e.preventDefault();
      setIsOpen(false);
      setActiveIndex(-1);
    }
  };

  // Scroll active option into view
  useEffect(() => {
    if (isOpen && activeIndex >= 0 && listboxRef.current) {
      const activeEl = listboxRef.current.children[activeIndex] as HTMLElement | undefined;
      if (typeof activeEl?.scrollIntoView === "function") {
        activeEl.scrollIntoView({ block: "nearest" });
      }
    }
  }, [isOpen, activeIndex]);

  const isInputValid = isValid ?? (required ? Boolean(value.trim()) : true);

  return (
    <div
      ref={containerRef}
      className={`searchable-select ${className} ${disabled ? "searchable-select--disabled" : ""}`}
      style={{ position: "relative", width: "100%" }}
    >
      <label
        htmlFor={id}
        className="searchable-select__label"
        style={{
          display: "block",
          fontSize: "0.75rem",
          fontWeight: 800,
          marginBottom: "0.2rem",
          color: "var(--ink, #1b1b2f)",
        }}
      >
        {label} {required && <span aria-hidden="true">*</span>}
      </label>

      <div
        className={`searchable-select__control ${
          isTouched ? (isInputValid ? "input--valid" : "input--invalid") : ""
        }`}
        style={{
          position: "relative",
          display: "flex",
          alignItems: "center",
          backgroundColor: disabled ? "#f1f1f1" : "#ffffff",
          border: "3px solid var(--ink, #1b1b2f)",
          borderRadius: "var(--radius-md, 0.875rem)",
          boxShadow: isOpen ? "4px 4px 0 var(--ink, #1b1b2f)" : "3px 3px 0 var(--ink, #1b1b2f)",
          transition: "all 120ms ease",
        }}
      >
        <input
          ref={inputRef}
          id={id}
          type="text"
          role="combobox"
          aria-expanded={isOpen}
          aria-haspopup="listbox"
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={
            isOpen && activeIndex >= 0 ? `${id}-option-${activeIndex}` : undefined
          }
          aria-invalid={isTouched ? !isInputValid : undefined}
          required={required}
          disabled={disabled}
          value={query}
          placeholder={placeholder}
          onFocus={() => {
            if (!disabled) setIsOpen(true);
          }}
          onChange={(e) => {
            const next = e.target.value;
            setQuery(next);
            setIsOpen(true);
            setActiveIndex(0);
            if (allowCustomInput) {
              onChange(next);
            }
          }}
          onKeyDown={handleKeyDown}
          style={{
            flex: 1,
            width: "100%",
            border: "none",
            outline: "none",
            backgroundColor: "transparent",
            padding: "0.5rem 0.65rem",
            fontSize: "0.875rem",
            fontWeight: 700,
            color: "var(--ink, #1b1b2f)",
            fontFamily: "inherit",
          }}
        />

        <div
          style={{
            display: "inline-flex",
            alignItems: "center",
            gap: "0.25rem",
            paddingRight: "0.5rem",
          }}
        >
          {Boolean(value || query) && !disabled && (
            <button
              type="button"
              aria-label="Hapus pilihan"
              onClick={handleClear}
              style={{
                background: "none",
                border: "none",
                cursor: "pointer",
                padding: "0.15rem 0.35rem",
                fontSize: "0.85rem",
                fontWeight: 900,
                color: "var(--ink-soft, #666)",
                borderRadius: "4px",
              }}
            >
              ✖
            </button>
          )}

          <button
            type="button"
            aria-label={isOpen ? "Tutup daftar pilihan" : "Buka daftar pilihan"}
            tabIndex={-1}
            disabled={disabled}
            onClick={() => {
              if (!disabled) {
                setIsOpen((prev) => !prev);
                inputRef.current?.focus();
              }
            }}
            style={{
              background: "none",
              border: "none",
              cursor: disabled ? "not-allowed" : "pointer",
              padding: "0.15rem 0.25rem",
              fontSize: "0.75rem",
              fontWeight: 900,
              color: "var(--ink, #1b1b2f)",
              transform: isOpen ? "rotate(180deg)" : "rotate(0deg)",
              transition: "transform 150ms ease",
            }}
          >
            ▼
          </button>
        </div>
      </div>

      {/* Dropdown Menu */}
      {isOpen && !disabled && (
        <ul
          ref={listboxRef}
          id={listboxId}
          role="listbox"
          aria-label={label}
          className="searchable-select__menu"
          style={{
            position: "absolute",
            top: "calc(100% + 4px)",
            left: 0,
            right: 0,
            maxHeight: "220px",
            overflowY: "auto",
            backgroundColor: "#ffffff",
            border: "2px solid var(--ink, #1b1b2f)",
            borderRadius: "var(--radius-md, 0.875rem)",
            boxShadow: "3px 3px 0 var(--ink, #1b1b2f)",
            zIndex: 1000,
            margin: 0,
            padding: "0.25rem 0",
            listStyle: "none",
          }}
        >
          {filteredOptions.length > 0 ? (
            filteredOptions.map((opt, idx) => {
              const isSelected = opt.value === value || opt.label === value;
              const isActive = idx === activeIndex;
              return (
                <li
                  key={`${opt.value}-${idx}`}
                  id={`${id}-option-${idx}`}
                  role="option"
                  aria-selected={isSelected}
                  onClick={() => handleSelectOption(opt)}
                  onMouseEnter={() => setActiveIndex(idx)}
                  className={`searchable-select__item ${
                    isSelected ? "searchable-select__item--selected" : ""
                  } ${isActive ? "searchable-select__item--active" : ""}`}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                    padding: "0.45rem 0.75rem",
                    fontSize: "0.8125rem",
                    fontWeight: isSelected ? 800 : 600,
                    cursor: "pointer",
                    backgroundColor: isActive
                      ? "var(--lemon, #ffd23f)"
                      : isSelected
                        ? "#f0fdf4"
                        : "transparent",
                    color: "var(--ink, #1b1b2f)",
                    borderBottom: "1px solid #f0f0f0",
                  }}
                >
                  <span style={{ display: "inline-flex", alignItems: "center", gap: "0.35rem" }}>
                    {isSelected && <span aria-hidden="true">✓</span>}
                    {opt.label}
                  </span>
                  {opt.badge && (
                    <span
                      className="badge badge--lemon"
                      style={{
                        fontSize: "0.7rem",
                        padding: "0.1rem 0.4rem",
                        borderRadius: "999px",
                        border: "1px solid var(--ink, #1b1b2f)",
                      }}
                    >
                      {opt.badge}
                    </span>
                  )}
                </li>
              );
            })
          ) : (
            <li
              style={{
                padding: "0.75rem",
                textAlign: "center",
                fontSize: "0.8125rem",
                color: "#666",
              }}
            >
              <div>Tidak ditemukan hasil untuk "{query}"</div>
              {allowCustomInput && query.trim() && (
                <button
                  type="button"
                  className="btn btn--lemon"
                  style={{
                    marginTop: "0.5rem",
                    fontSize: "0.75rem",
                    padding: "0.25rem 0.6rem",
                    minHeight: "auto",
                  }}
                  onClick={() => {
                    onChange(query.trim());
                    setIsOpen(false);
                  }}
                >
                  Gunakan "{query.trim()}"
                </button>
              )}
            </li>
          )}
        </ul>
      )}

      {/* Validation Feedback & Status Badge */}
      {isTouched && (
        <div
          role="status"
          aria-live="polite"
          style={{ marginTop: "0.25rem" }}
          className={isInputValid ? "field-feedback--valid" : "field-feedback--invalid"}
        >
          {isInputValid
            ? `✓ ${label} ${value ? "Terpilih" : "Valid"}`
            : `✕ ${feedbackText || `${label} wajib diisi`}`}
        </div>
      )}
    </div>
  );
};
