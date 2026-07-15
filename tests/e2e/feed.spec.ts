import { test, expect } from "@playwright/test";

// Consumer-side feed: the interaction -> profile -> ranking closed loop, driven
// through the real UI. Runs after admin-flows.spec.ts (single worker), so the
// active strategy may be anything — assertions here are strategy-independent.

test.beforeEach(async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "For You Feed" }).click();
});

test("Feed renders a ranked video card with reason badge", async ({ page }) => {
  await expect(page.getByTestId("feed-frame")).toBeVisible();
  await expect(page.getByText(/Rank #1 \/ \d+/)).toBeVisible();
  // Every card carries an explainable reason from the ranking strategy.
  await expect(page.getByText(/interest_match|globally_trending|recency/).first()).toBeVisible();
});

test("Liking a video posts an interaction and updates the live profile", async ({ page }) => {
  // user_new starts cold (no seeded interactions).
  await page.getByRole("button", { name: "user_new", exact: true }).click();
  await expect(page.getByText(/Rank #1 \/ \d+/)).toBeVisible();

  await page.getByTestId("like-button").click();

  // The event log records the POST and the profile panel gains an interest tag.
  await expect(page.getByTestId("event-log")).toContainText("like");
  await expect(page.getByTestId("profile-tags")).toBeVisible();
});

test("Re-rank Feed reloads the ranking from the current profile", async ({ page }) => {
  await page.getByTestId("rerank-button").click();
  await expect(page.getByTestId("event-log")).toContainText("feed re-ranked");
  await expect(page.getByText(/Rank #1 \/ \d+/)).toBeVisible();
});
