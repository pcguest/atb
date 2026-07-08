import { dirname } from "node:path";
import { fileURLToPath } from "node:url";

import tseslint from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";

// import.meta.dirname requires Node 20.11+; package.json supports >=18.
const configDir = dirname(fileURLToPath(import.meta.url));

export default [
  {
    ignores: ["dist/**", "src/eventTypes_generated.ts"],
  },
  {
    files: ["src/**/*.ts"],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        project: "./tsconfig.typecheck.json",
        tsconfigRootDir: configDir,
      },
    },
    plugins: {
      "@typescript-eslint": tseslint,
    },
    rules: {
      ...tseslint.configs.recommended.rules,
    },
  },
];
