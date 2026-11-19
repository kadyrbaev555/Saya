import { defineConfig } from "@playwright/test";

const port = Number(process.env.SAYA_TEST_WEB_PORT || 3000);
const baseURL = "http://localhost:" + port;

export default defineConfig({
  testDir: "./tests",
  // The integration suite shares one local gateway and its per-IP request budget.
  workers: 1,
  timeout: 60000,
  use: {
    launchOptions: { slowMo: 350 },
    baseURL,
    channel: process.env.SAYA_BROWSER_CHANNEL || "chromium",
    trace: "off",
    screenshot: "off",
  },
  webServer: {
    command: "npm run build && npm run preview -- --strictPort --port " + port,
    url: baseURL,
    reuseExistingServer: false,
    timeout: 60000,
  },
});
