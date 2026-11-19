import { defineConfig } from "vite";

export default defineConfig({
  server: {
    port: 3000,
    strictPort: true,
    proxy: {
      "/api": process.env.SAYA_GATEWAY_URL || "http://127.0.0.1:8080",
      "/connection/websocket": {
        target: process.env.SAYA_CENTRIFUGO_URL || "ws://127.0.0.1:8000",
        ws: true,
      },
    },
  },
});
