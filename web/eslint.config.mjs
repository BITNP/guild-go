import { so1ve } from "@so1ve/eslint-config";

export default so1ve(
  {},
  {
    files: ["**/*.html"],
    rules: {
      "html/use-baseline": "off",
    },
  },
  {
    rules: {
      "style/padding-line-between-statements": [
        "error",
        {
          blankLine: "always",
          prev: "*",
          next: ["return", "continue", "break", "throw", "debugger"],
        },
        {
          blankLine: "always",
          prev: "function",
          next: "*",
        },
        {
          blankLine: "always",
          prev: "*",
          next: "function",
        },
      ],
    },
  },
);
