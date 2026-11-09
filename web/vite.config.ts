import { URL, fileURLToPath } from "node:url";

import * as Lucide from "@lucide/vue";
import vue from "@vitejs/plugin-vue";
import AutoImport from "unplugin-auto-import/vite";
import { NaiveUiResolver } from "unplugin-vue-components/resolvers";
import type { ComponentResolver } from "unplugin-vue-components/types";
import Components from "unplugin-vue-components/vite";
import { defineConfig } from "vite";

const lucideComponents = new Set(Object.keys(Lucide));
const lucideResolver: ComponentResolver = (name) =>
  lucideComponents.has(name) ? { name, from: "@lucide/vue" } : undefined;
const routerResolver: ComponentResolver = (name) =>
  name === "RouterLink" || name === "RouterView"
    ? { name, from: "vue-router" }
    : undefined;

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      dts: "src/auto-imports.d.ts",
      dtsMode: "overwrite",
      dirs: ["src/api/index.ts", "src/composables", "src/lib"],
      imports: [
        "vue",
        "vue-router",
        {
          "naive-ui": [
            "createDiscreteApi",
            "darkTheme",
            "dateZhCN",
            "NEllipsis",
            "NFlex",
            "NTag",
            "NText",
            "useThemeVars",
            "zhCN",
          ],
          "vue": ["cloneVNode"],
          "@lucide/vue": ["Check", "Circle", "LogOut", "UserRound"],
          "vue-router": [
            "NavigationFailureType",
            "RouterLink",
            "createRouter",
            "createWebHistory",
            "isNavigationFailure",
          ],
        },
      ],
      vueTemplate: true,
    }),
    Components({
      deep: true,
      dirs: ["src/components"],
      dts: "src/components.d.ts",
      extensions: ["vue"],
      resolvers: [NaiveUiResolver(), lucideResolver, routerResolver],
    }),
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: false,
      },
    },
  },
  build: {
    outDir: "../internal/web/dist",
    emptyOutDir: true,
  },
});
