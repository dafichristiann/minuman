import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

/**
 * ESLint flat config (ESLint 9).
 *
 * Aturan yang ditegakkan:
 *   - TypeScript strict-aware, tanpa `any` implisit
 *   - Aturan Hooks React (mencegah bug urutan hook)
 *   - Larangan `console.*` di kode aplikasi: gunakan logger, bukan console
 *     (jejak debug yang tertinggal di produksi adalah sumber kebocoran data §13.1)
 *   - Larangan `dangerouslySetInnerHTML`: satu-satunya jalur XSS di React
 */
export default tseslint.config(
  {
    ignores: ["dist", "coverage", "node_modules", "playwright-report", "test-results"],
  },
  {
    files: ["**/*.{ts,tsx}"],
    extends: [js.configs.recommended, ...tseslint.configs.recommendedTypeChecked],
    languageOptions: {
      ecmaVersion: 2022,
      globals: { ...globals.browser, ...globals.es2022 },
      parserOptions: {
        project: ["./tsconfig.app.json", "./tsconfig.node.json"],
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],

      // Kualitas & keamanan
      "no-console": ["error", { allow: ["warn", "error"] }],
      "no-debugger": "error",
      "no-alert": "error",
      "no-restricted-syntax": [
        "error",
        {
          selector: "JSXAttribute[name.name='dangerouslySetInnerHTML']",
          message:
            "dangerouslySetInnerHTML dilarang (§13.1). Render teks sebagai children agar React meng-escape.",
        },
        {
          selector: "MemberExpression[object.name='localStorage']",
          message:
            "Token/sesi tidak boleh disimpan di localStorage (§13.1). Gunakan cookie httpOnly.",
        },
      ],
      eqeqeq: ["error", "always", { null: "ignore" }],
      "prefer-const": "error",
      "no-var": "error",

      // TypeScript
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
      "@typescript-eslint/consistent-type-imports": [
        "error",
        { prefer: "type-imports", fixStyle: "inline-type-imports" },
      ],
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/no-misused-promises": [
        "error",
        { checksVoidReturn: { attributes: false } },
      ],
    },
  },
  {
    // Berkas JS murni (eslint.config.js) tidak ikut type-checking.
    files: ["**/*.js"],
    extends: [js.configs.recommended],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: "module",
      globals: { ...globals.node },
    },
  },
  {
    // Berkas uji & konfigurasi boleh lebih longgar.
    files: ["**/*.{test,spec}.{ts,tsx}", "vite.config.ts"],
    rules: {
      "@typescript-eslint/require-await": "off",
      "@typescript-eslint/no-base-to-string": "off",
      "@typescript-eslint/no-unsafe-assignment": "off",
      "@typescript-eslint/no-unsafe-member-access": "off",
      "@typescript-eslint/no-unsafe-argument": "off",
      "@typescript-eslint/no-unsafe-call": "off",
      "no-console": "off",
    },
  },
);
