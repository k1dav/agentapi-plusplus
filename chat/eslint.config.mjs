// ESLint v9 flat config for the chat app.
// Pairs eslint-config-next with the Storybook plugin so `next lint` runs
// deterministically in CI and locally (the previous setup prompted for an
// initial config and never produced one).
import { FlatCompat } from "@eslint/eslintrc";
import storybookRecommended from "eslint-plugin-storybook/dist/configs/flat/recommended.mjs";
import { fileURLToPath } from "node:url";
import { dirname } from "node:path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const compat = new FlatCompat({ baseDirectory: __dirname });

export default [
  {
    ignores: [
      ".next/**",
      "node_modules/**",
      "out/**",
      "storybook-static/**",
      "next-env.d.ts",
    ],
  },
  ...compat.extends("next/core-web-vitals", "next/typescript"),
  ...storybookRecommended,
];