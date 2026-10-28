import { expect, test } from "@playwright/test";

test("a person opens the Home page and sees the name of the application", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("heading", { level: 1, name: "go-react-template" })).toBeVisible();
});
