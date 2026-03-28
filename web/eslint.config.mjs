import nextCoreWebVitals from "eslint-config-next/core-web-vitals";
import prettierConfig from "eslint-config-prettier";

const eslintConfig = [
  {
    ignores: ["coverage/**"],
  },
  ...nextCoreWebVitals,
  prettierConfig,
];

export default eslintConfig;
