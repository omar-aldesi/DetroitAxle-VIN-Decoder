import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],

  build: {
    chunkSizeWarningLimit: 700,
    rollupOptions: {
      output: {
        manualChunks: {
          // React runtime — changes almost never, long cache TTL
          "vendor-react": ["react", "react-dom", "react-router-dom"],
          // Data-fetching layer
          "vendor-query": ["@tanstack/react-query"],
          // HTTP client
          "vendor-axios": ["axios"],
          // Icon library (large)
          "vendor-icons": ["lucide-react"],
        },
      },
    },
  },

  server: {
    port: 5173,
    host: true,
    allowedHosts: [
      "mike-pointed-busy-quotes.trycloudflare.com",
      "returned-mississippi-powell-nano.trycloudflare.com",
      "technician-shareholders-reno-robert.trycloudflare.com",
    ],
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
