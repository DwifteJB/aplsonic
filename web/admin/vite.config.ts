import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// proxy
export default defineConfig({
  base: "/admin/",
  plugins: [react(), tailwindcss()],
  build: {
    outDir: "../../src/serve/admin/dist",
    emptyOutDir: true,
  },
  server: {
    // dev: forward API calls to the running Go server's web_port (configuration.yml)
    proxy: {
      "/admin/api": "http://localhost:4534",
    },
  },
});
