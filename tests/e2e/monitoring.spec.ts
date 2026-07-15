import { test, expect } from "@playwright/test";

// Live observability dashboard + one-click demo traffic. Runs last (single
// worker, alphabetical order) because the traffic generator mutates profiles.

test.beforeEach(async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Live Monitoring" }).click();
});

test("Monitoring page shows stat cards, charts and per-service table", async ({ page }) => {
  await expect(page.getByText("Demo Traffic Generator")).toBeVisible();
  await expect(page.getByText("Throughput (edge)")).toBeVisible();
  await expect(page.getByText("Latency p99 / p50")).toBeVisible();
  await expect(page.getByText("Requests per second")).toBeVisible();

  // All three logical services report UP through the aggregated metrics feed.
  const table = page.getByTestId("service-table");
  await expect(table).toContainText("user");
  await expect(table).toContainText("content");
  await expect(table).toContainText("recommendation");
  await expect(table.getByText("UP", { exact: true })).toHaveCount(3);
});

test("One-click burst runs a measured load wave", async ({ page }) => {
  await page.getByTestId("burst-button").click();
  // The burst response doubles as a mini load-test report.
  await expect(page.getByTestId("burst-result")).toContainText("rps", { timeout: 30_000 });
  await expect(page.getByTestId("burst-result")).toContainText("p99");
});

test("Continuous traffic toggle drives live counters", async ({ page }) => {
  await page.getByTestId("traffic-toggle").click(); // Start
  await expect(page.getByTestId("traffic-toggle")).toContainText("Stop Traffic");

  // Sent counter climbs as server-side synthetic load flows.
  await expect(async () => {
    const text = await page.getByTestId("traffic-sent").innerText();
    const sent = Number(/sent:\s*(\d+)/.exec(text)?.[1] ?? 0);
    expect(sent).toBeGreaterThan(0);
  }).toPass({ timeout: 15_000 });

  await page.getByTestId("traffic-toggle").click(); // Stop
  await expect(page.getByTestId("traffic-toggle")).toContainText("Start Traffic");
});
