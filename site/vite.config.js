import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [tailwindcss()],
  base: "/parkranger/",
  build: {
    outDir: "dist",
  },
  publicDir: "public",
});
