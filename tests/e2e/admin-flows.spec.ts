import { test, expect } from "@playwright/test";

// Key user stories for the recommendation admin platform, driven end to end
// through the real browser UI (mock API mode for determinism).

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test("Dashboard shows live health metrics and microservice topology", async ({ page }) => {
  // Metric cards are populated from GET /api/v1/health.
  await expect(page.getByText("1250 RPS")).toBeVisible();
  await expect(page.getByText("32 ms")).toBeVisible();
  // Microservice topology lists the core services.
  await expect(page.getByText("Recommendation Core (Go)")).toBeVisible();
  await expect(page.getByText("Management Dashboard (React)")).toBeVisible();
  await expect(page.getByText("PostgreSQL Cluster")).toBeVisible();
});

test("Admin can log in and the session is reflected in the UI", async ({ page }) => {
  await page.getByRole("button", { name: "Admin Access" }).click();
  await expect(page.getByRole("button", { name: "Admin Logged In" })).toBeVisible();
  await expect(page.getByText("Authenticated with TikTok Global IAM")).toBeVisible();
});

test("Simulator returns a ranked recommendation feed for a user_id", async ({ page }) => {
  await page.getByRole("button", { name: "Demo Simulator" }).click();

  const input = page.getByPlaceholder("Enter UUID or Username...");
  await expect(input).toHaveValue("user_123");

  await page.getByRole("button", { name: "Fetch Recommendations" }).click();

  // Ranked video cards render with title and an explainable reason badge.
  // (Target the heading to avoid also matching the raw-JSON <pre> blob.)
  await expect(page.getByRole("heading", { name: "Top Tech 2026" })).toBeVisible();
  await expect(page.getByText("interest_match_tech").first()).toBeVisible();
  // Raw response panel proves the API round-trip.
  await expect(page.getByText("Raw Response (v1)")).toBeVisible();
});

test("Admin updates the ranking strategy and sees it in the deployment log", async ({ page }) => {
  await page.getByRole("button", { name: "Algorithm Config" }).click();

  // Switch the active strategy and deploy (handler logs in automatically if needed).
  await page.getByRole("combobox").selectOption("chronological");
  await page.getByRole("button", { name: "Deploy to Production" }).click();

  // Success feedback + persisted deployment log entry.
  await expect(page.getByText(/deployed to Ranking Shards successfully/i)).toBeVisible();
  await expect(page.getByText("STRATEGY_SET: CHRONOLOGICAL").first()).toBeVisible();
});
