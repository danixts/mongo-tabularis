import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const entries: Record<string, string> = {
  "filter-bar": "src/filter-bar.tsx",
};

const target = process.env.BUILD_ENTRY ?? "filter-bar";

export default defineConfig({
  plugins: [react()],
  define: {
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
  build: {
    outDir: "../ui",
    emptyOutDir: false,
    minify: "esbuild",
    lib: {
      entry: entries[target],
      formats: ["iife"],
      name: "__tabularis_plugin__",
      fileName: () => `${target}.js`,
    },
    rollupOptions: {
      external: ["react", "react/jsx-runtime"],
      output: {
        globals: {
          react: "React",
          "react/jsx-runtime": "ReactJSXRuntime",
        },
      },
    },
  },
});
