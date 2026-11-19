import { defineConfig } from "@playwright/test";

const port = Number(process.env.SAYA_TEST_WEB_PORT || 3000);
const baseURL = "http://localhost:" + port;

export default defineConfig({
  testDir: "./tests",
  use: {
    baseURL,
    channel: process.env.SAYA_BROWSER_CHANNEL || "chromium",
    trace: "off",
    screenshot: "off",
  },
  webServer: {
    command: "npm run dev -- --port " + port,
    url: baseURL,
    reuseExistingServer: false,
  },
});
