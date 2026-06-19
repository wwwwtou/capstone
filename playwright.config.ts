import { defineConfig, devices } from "@playwright/test";

// End-to-end tests drive the real React UI against the app's deterministic
// in-memory mock API (server.ts MOCK mode — no GATEWAY_URL, no Docker needed),
// so the key admin user stories are exercised browser -> UI -> API in one process.
export default defineConfig({
  testDir: "./tests/e2e",
  // The mock API holds shared in-memory state (config + deployment history),
  // so tests run serially in a single worker to stay deterministic.
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  expect: { timeout: 10_000 },
  use: {
    baseURL: "http://localhost:3101",
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: "npm run dev",
    url: "http://localhost:3101",
    // Dedicated port + forced MOCK mode (empty GATEWAY_URL) so E2E is
    // deterministic and never reuses a stray proxy-mode dev server on :3000.
    env: { GATEWAY_URL: "", PORT: "3101" },
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
