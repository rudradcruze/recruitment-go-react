import { expect, test } from "@playwright/test";

import { createQuestion, localInput, logInAsAdmin, uniqueId } from "./support";

test("an Admin schedules an Exam, a Candidate takes it through the Exam link, and the Admin reads the result", async ({
  page,
  browser,
}) => {
  const id = uniqueId();
  const question1 = `What is 2 + 2? (${id})`;
  const question2 = `What is the capital of France? (${id})`;
  const title = `Exam ${id}`;
  const name = `Ann ${id}`;
  const emailAddress = `ann-${id}@example.com`;

  // The Admin logs in and makes 2 Questions.
  await logInAsAdmin(page);
  await createQuestion(page, question1, [`4 (${id})`, `5 (${id})`]);
  await expect(
    page.getByRole("row", { name: new RegExp(question1.replace(/[()+?]/g, "\\$&")) }),
  ).toBeVisible();
  await createQuestion(page, question2, [`Paris (${id})`, `Rome (${id})`]);
  await expect(page.getByRole("heading", { level: 1, name: "Questions" })).toBeVisible();

  // The Admin makes an Exam with an open Exam window, and copies the Exam link.
  await page.goto("/admin/exams/new");
  await page.getByLabel("Title").fill(title);
  await page.getByLabel("Open time").fill(localInput(new Date(Date.now() - 60 * 60 * 1000)));
  await page.getByLabel("Close time").fill(localInput(new Date(Date.now() + 2 * 60 * 60 * 1000)));
  await page.getByLabel(question1).check();
  await page.getByLabel(question2).check();
  await page.getByRole("button", { name: "Save Exam" }).click();
  const examRow = page.getByRole("row", { name: new RegExp(title) });
  await expect(examRow).toBeVisible();
  await examRow.getByRole("link", { name: "Edit" }).click();
  const link = (await page.locator("code").textContent()) ?? "";
  expect(link).toMatch(/\/e\/[0-9A-Za-z]{22}$/);

  // A Candidate, in a new browser context, opens the link, and answers 1 Question right and 1 Question wrong.
  const candidateContext = await browser.newContext();
  const candidate = await candidateContext.newPage();
  await candidate.goto(link);
  await expect(candidate.getByRole("heading", { level: 1, name: title })).toBeVisible();
  await candidate.getByLabel("Name").fill(name);
  await candidate.getByLabel("Email address").fill(emailAddress);
  await candidate.getByRole("button", { name: "Start" }).click();
  await expect(candidate.getByRole("group", { name: question1 })).toBeVisible();
  await expect(candidate.getByRole("timer")).toBeVisible();
  await candidate.getByLabel(`4 (${id})`).check();
  await expect(candidate.getByText("The Answer is saved.")).toBeVisible();
  await candidate.getByLabel(`Rome (${id})`).check();
  await candidate.getByRole("button", { name: "Submit" }).click();
  await expect(candidate.getByText("Your Score is 1 of 2.")).toBeVisible();

  // The Admin opens the results and sees the row of the Candidate.
  await page.getByRole("link", { name: "Results" }).click();
  const resultRow = page.getByRole("row", { name: new RegExp(name) });
  await expect(resultRow).toBeVisible();
  await expect(resultRow).toContainText(emailAddress);
  await expect(resultRow).toContainText("Submitted");
  await expect(resultRow).toContainText("1 of 2");

  // The Candidate opens the link again with the same email address, and cannot start a second Attempt.
  await candidate.goto(link);
  await candidate.getByLabel("Name").fill(name);
  await candidate.getByLabel("Email address").fill(emailAddress.toUpperCase());
  await candidate.getByRole("button", { name: "Start" }).click();
  await expect(candidate.getByRole("alert")).toContainText("This Exam is already submitted");
  await candidateContext.close();
});

test("a Candidate who refreshes the Attempt page keeps the saved Answers, and sees the message for a finished Attempt after the Submit", async ({
  page,
  browser,
}) => {
  const id = uniqueId();
  await logInAsAdmin(page);
  await createQuestion(page, `Question A (${id})`, [`Right (${id})`, `Wrong (${id})`]);
  await expect(page.getByRole("heading", { level: 1, name: "Questions" })).toBeVisible();
  const questions = await page.request.get("/api/v1/admin/questions");
  const created = (await questions.json()) as Array<{ id: string; text: string }>;
  const questionId = created.find((q) => q.text === `Question A (${id})`)?.id;
  expect(questionId).toBeDefined();
  const exam = await page.request.post("/api/v1/admin/exams", {
    data: {
      title: `Exam ${id}`,
      questionIds: [questionId],
      opensAt: new Date(Date.now() - 3600_000).toISOString(),
      closesAt: new Date(Date.now() + 3600_000).toISOString(),
      timeLimitMinutes: 30,
    },
  });
  expect(exam.status()).toBe(201);
  const { linkToken } = (await exam.json()) as { linkToken: string };

  const context = await browser.newContext();
  const candidate = await context.newPage();
  await candidate.goto(`/e/${linkToken}`);
  await candidate.getByLabel("Name").fill("Bob");
  await candidate.getByLabel("Email address").fill(`bob-${id}@example.com`);
  await candidate.getByRole("button", { name: "Start" }).click();
  await candidate.getByLabel(`Wrong (${id})`).check();
  await expect(candidate.getByText("The Answer is saved.")).toBeVisible();

  await candidate.reload();
  await expect(candidate.getByLabel(`Wrong (${id})`)).toBeChecked();

  await candidate.getByRole("button", { name: "Submit" }).click();
  await expect(candidate.getByText("Your Score is 0 of 1.")).toBeVisible();
  await candidate.reload();
  await expect(candidate.getByText("This Exam is already submitted.")).toBeVisible();
  await expect(candidate.getByText(/Score/)).toHaveCount(0);
  await context.close();
});

test("a person without a session who opens an Admin page goes to the login page", async ({
  page,
}) => {
  await page.goto("/admin/questions");

  await expect(page).toHaveURL(/\/admin\/login$/);
  await expect(page.getByRole("heading", { level: 1, name: "Log in" })).toBeVisible();
});

test("an Admin who enters a wrong password sees the message of the API", async ({ page }) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email address").fill(`nobody-${uniqueId()}@example.com`);
  await page.getByLabel("Password").fill("not the password");
  await page.getByRole("button", { name: "Log in" }).click();

  await expect(page.getByRole("alert")).toHaveText(
    "The email address or the password is not correct. Enter them again.",
  );
});

test("an Admin logs out and cannot open an Admin page after it", async ({ page }) => {
  await logInAsAdmin(page);
  await expect(page.getByRole("heading", { level: 1, name: "Admin" })).toBeVisible();

  await page.getByRole("button", { name: "Log out" }).click();
  await expect(page).toHaveURL(/\/admin\/login$/);
  await page.goto("/admin/exams");

  await expect(page).toHaveURL(/\/admin\/login$/);
});
