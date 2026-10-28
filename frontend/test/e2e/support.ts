import { randomUUID } from "node:crypto";

import { expect, type Page } from "@playwright/test";

// compose.e2e.yaml gives these values to the migrate service, which makes this Admin. They are not secret.
export const adminEmail = "admin@example.com";
export const adminPassword = "e2e-admin-password";

// uniqueId gives a short value that is new in each call. Each test makes its own data with it.
export function uniqueId(): string {
  return randomUUID().slice(0, 8);
}

// localInput gives the value of a datetime-local field for a time. The field uses the timezone of the browser,
// and the browser of the tests has the timezone of this machine.
export function localInput(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export async function logInAsAdmin(page: Page) {
  await page.goto("/admin/login");
  await page.getByLabel("Email address").fill(adminEmail);
  await page.getByLabel("Password").fill(adminPassword);
  await page.getByRole("button", { name: "Log in" }).click();
  // Wait until the login ends. A later page.goto cancels a request that is still open, and the session is then not set.
  await expect(page.getByRole("heading", { level: 1, name: "Admin" })).toBeVisible();
}

// createQuestion makes a Question in the Admin area. The first Option is correct.
export async function createQuestion(page: Page, text: string, options: [string, string]) {
  await page.goto("/admin/questions/new");
  await page.getByLabel("Question text").fill(text);
  await page.getByLabel("Option 1", { exact: true }).fill(options[0]);
  await page.getByLabel("Option 2", { exact: true }).fill(options[1]);
  await page.getByLabel("Option 1 is correct").check();
  await page.getByRole("button", { name: "Save Question" }).click();
}
