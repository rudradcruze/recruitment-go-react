import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, http, HttpResponse } from "msw";
import { describe, expect, it } from "vite-plus/test";

import { admin, type Admin } from "../../fixtures/admin";
import {
  attemptResult,
  exam,
  examSummary,
  type AttemptResult,
  type Exam,
  type ExamRequest,
  type ExamSummary,
} from "../../fixtures/exam";
import { problem } from "../../fixtures/problem";
import { question, type Question } from "../../fixtures/question";
import { server } from "../../mocks/server";
import {
  examAttemptsUrl,
  examLinkUrl,
  examUrl,
  examsUrl,
  questionsUrl,
  sessionUrl,
} from "../../mocks/urls";
import { renderApp } from "../../render";

function loggedIn() {
  server.use(http.get<never, never, Admin>(sessionUrl, () => HttpResponse.json(admin())));
}

function examListHandler(exams: ExamSummary[]) {
  return http.get<never, never, ExamSummary[]>(examsUrl, () => HttpResponse.json(exams));
}

function questionListHandler(questions: Question[]) {
  return http.get<never, never, Question[]>(questionsUrl, () => HttpResponse.json(questions));
}

const local = (iso: string) => new Date(iso).toLocaleString();

describe("the Exam list page", () => {
  it("shows each Exam with its Exam window in local time, its Time limit and its number of Questions", async () => {
    loggedIn();
    const summary = examSummary({ title: "Go basics", timeLimitMinutes: 45, questionCount: 3 });
    server.use(examListHandler([summary]));
    renderApp("/admin/exams");

    const row = await screen.findByRole("row", { name: /Go basics/ });
    expect(row).toHaveTextContent(local(summary.opensAt));
    expect(row).toHaveTextContent(local(summary.closesAt));
    expect(row).toHaveTextContent("45 minutes");
    expect(within(row).getByRole("cell", { name: "3" })).toBeInTheDocument();
    expect(within(row).getByRole("link", { name: "Edit" })).toHaveAttribute(
      "href",
      `/admin/exams/${summary.id}`,
    );
    expect(within(row).getByRole("link", { name: "Results" })).toHaveAttribute(
      "href",
      `/admin/exams/${summary.id}/results`,
    );
    expect(screen.getByRole("link", { name: "New Exam" })).toHaveAttribute(
      "href",
      "/admin/exams/new",
    );
    expect(document.title).toBe("Exams - go-react-template");
  });

  it("is reachable from the link in the Admin area", async () => {
    loggedIn();
    server.use(examListHandler([]));
    renderApp("/admin");
    const person = userEvent.setup();

    await person.click(await screen.findByRole("link", { name: "Exams" }));

    expect(await screen.findByRole("heading", { level: 1, name: "Exams" })).toBeInTheDocument();
    await screen.findByText(/There is no Exam/);
  });

  it("shows an empty state when there is no Exam", async () => {
    loggedIn();
    server.use(examListHandler([]));
    renderApp("/admin/exams");

    expect(await screen.findByText(/There is no Exam/)).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("asks to confirm a delete, deletes the Exam, and reloads the list", async () => {
    loggedIn();
    const first = examSummary({ title: "First exam" });
    const second = examSummary({ title: "Second exam" });
    let remaining = [first, second];
    const deleted: string[] = [];
    server.use(
      http.get<never, never, ExamSummary[]>(examsUrl, () => HttpResponse.json(remaining)),
      http.delete<{ id: string }>(examUrl, ({ params }) => {
        deleted.push(params.id);
        remaining = remaining.filter((e) => e.id !== params.id);
        return new HttpResponse(null, { status: 204 });
      }),
    );
    renderApp("/admin/exams");
    const person = userEvent.setup();
    const row = await screen.findByRole("row", { name: /First exam/ });

    await person.click(within(row).getByRole("button", { name: "Delete" }));
    expect(deleted).toEqual([]);
    await person.click(within(row).getByRole("button", { name: "Confirm delete" }));

    expect(await screen.findByText("The Exam is deleted.")).toBeInTheDocument();
    await screen.findByRole("row", { name: /Second exam/ });
    expect(screen.queryByRole("row", { name: /First exam/ })).not.toBeInTheDocument();
    expect(deleted).toEqual([first.id]);
  });

  it("shows the detail of a 409 response when Candidates started the Exam", async () => {
    loggedIn();
    const detail =
      "Candidates started this Exam. Only the close time can change, and you cannot delete the Exam.";
    server.use(
      examListHandler([examSummary({ title: "First exam" })]),
      http.delete(examUrl, () => HttpResponse.json(problem(409, { detail }), { status: 409 })),
    );
    renderApp("/admin/exams");
    const person = userEvent.setup();
    const row = await screen.findByRole("row", { name: /First exam/ });

    await person.click(within(row).getByRole("button", { name: "Delete" }));
    await person.click(within(row).getByRole("button", { name: "Confirm delete" }));

    expect(await screen.findByText(detail)).toBeInTheDocument();
  });
});

describe("the new Exam page", () => {
  async function fillForm(person: ReturnType<typeof userEvent.setup>) {
    await person.type(screen.getByLabelText("Title"), "Go basics");
    await person.type(screen.getByLabelText("Open time"), "2026-11-01T09:00");
    await person.type(screen.getByLabelText("Close time"), "2026-11-01T17:00");
    await person.clear(screen.getByLabelText("Time limit in minutes"));
    await person.type(screen.getByLabelText("Time limit in minutes"), "45");
  }

  it("has a label for each field and a list of the Questions of the Question bank", async () => {
    loggedIn();
    server.use(
      questionListHandler([
        question({ text: "What is 2 + 2?" }),
        question({ text: "What is Go?" }),
      ]),
    );
    renderApp("/admin/exams/new");

    expect(await screen.findByRole("heading", { level: 1, name: "New Exam" })).toBeInTheDocument();
    expect(await screen.findByLabelText("What is 2 + 2?")).not.toBeChecked();
    expect(screen.getByLabelText("What is Go?")).toHaveAttribute("type", "checkbox");
    expect(screen.getByLabelText("Title")).toBeInTheDocument();
    expect(screen.getByLabelText("Open time")).toHaveAttribute("type", "datetime-local");
    expect(screen.getByLabelText("Close time")).toHaveAttribute("type", "datetime-local");
    expect(screen.getByLabelText("Time limit in minutes")).toHaveAttribute("type", "number");
    expect(screen.getByRole("button", { name: "Save Exam" })).toBeInTheDocument();
    expect(document.title).toBe("New Exam - go-react-template");
  });

  it("tells the Admin to create a Question when the Question bank is empty", async () => {
    loggedIn();
    server.use(questionListHandler([]));
    renderApp("/admin/exams/new");

    expect(await screen.findByText(/The Question bank has no Questions/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "New Question" })).toHaveAttribute(
      "href",
      "/admin/questions/new",
    );
  });

  it("keeps the Questions in the order that the Admin selects them, and shows them in a numbered list", async () => {
    loggedIn();
    const [a, b, c] = [
      question({ text: "Alpha?" }),
      question({ text: "Bravo?" }),
      question({ text: "Charlie?" }),
    ];
    server.use(questionListHandler([a, b, c]));
    renderApp("/admin/exams/new");
    const person = userEvent.setup();

    await person.click(await screen.findByLabelText("Charlie?"));
    await person.click(screen.getByLabelText("Alpha?"));
    await person.click(screen.getByLabelText("Bravo?"));
    await person.click(screen.getByLabelText("Alpha?"));

    const items = within(screen.getByRole("list", { name: "Selected Questions" })).getAllByRole(
      "listitem",
    );
    expect(items.map((item) => item.textContent)).toEqual(["Charlie?", "Bravo?"]);
  });

  it("sends the title, the Questions in order, the times in UTC and the Time limit, and opens the list", async () => {
    loggedIn();
    const [a, b] = [question({ text: "Alpha?" }), question({ text: "Bravo?" })];
    const received: ExamRequest[] = [];
    server.use(
      questionListHandler([a, b]),
      http.post<never, ExamRequest, Exam>(examsUrl, async ({ request }) => {
        received.push(await request.json());
        return HttpResponse.json(exam(), { status: 201 });
      }),
      examListHandler([]),
    );
    const { router } = renderApp("/admin/exams/new");
    const person = userEvent.setup();
    await screen.findByLabelText("Alpha?");

    await fillForm(person);
    await person.click(screen.getByLabelText("Bravo?"));
    await person.click(screen.getByLabelText("Alpha?"));
    await person.click(screen.getByRole("button", { name: "Save Exam" }));

    await screen.findByText(/There is no Exam/);
    expect(received).toEqual([
      {
        title: "Go basics",
        questionIds: [b.id, a.id],
        opensAt: new Date("2026-11-01T09:00").toISOString(),
        closesAt: new Date("2026-11-01T17:00").toISOString(),
        timeLimitMinutes: 45,
      },
    ]);
    expect(router.state.location.pathname).toBe("/admin/exams");
    expect(router.state.historyAction).toBe("REPLACE");
  });

  it("shows the detail of a 422 response and stays on the form with the values", async () => {
    loggedIn();
    const detail = "The Exam window is not valid. Give an open time that is before the close time.";
    server.use(
      questionListHandler([question({ text: "Alpha?" })]),
      http.post(examsUrl, () => HttpResponse.json(problem(422, { detail }), { status: 422 })),
    );
    const { router } = renderApp("/admin/exams/new");
    const person = userEvent.setup();
    await screen.findByLabelText("Alpha?");
    await person.type(screen.getByLabelText("Title"), "Go basics");

    await person.click(screen.getByRole("button", { name: "Save Exam" }));

    expect(await screen.findByText(detail)).toBeInTheDocument();
    expect(screen.getByLabelText("Title")).toHaveValue("Go basics");
    expect(router.state.location.pathname).toBe("/admin/exams/new");
  });

  it("does not send the form twice while it waits for the server", async () => {
    loggedIn();
    let requests = 0;
    server.use(
      questionListHandler([question({ text: "Alpha?" })]),
      http.post(examsUrl, async () => {
        requests += 1;
        await delay(100);
        return HttpResponse.json(exam(), { status: 201 });
      }),
      examListHandler([]),
    );
    renderApp("/admin/exams/new");
    const person = userEvent.setup();
    await screen.findByLabelText("Alpha?");

    await person.click(screen.getByRole("button", { name: "Save Exam" }));
    const button = screen.getByRole("button", { name: "Save Exam" });
    expect(button).toHaveAttribute("aria-disabled", "true");
    expect(button).toHaveFocus();
    await person.click(button);

    await screen.findByText(/There is no Exam/);
    expect(requests).toBe(1);
  });

  it("works with the keyboard only", async () => {
    loggedIn();
    const [a] = [question({ text: "Alpha?" })];
    const received: ExamRequest[] = [];
    server.use(
      questionListHandler([a]),
      http.post<never, ExamRequest, Exam>(examsUrl, async ({ request }) => {
        received.push(await request.json());
        return HttpResponse.json(exam(), { status: 201 });
      }),
      examListHandler([]),
    );
    renderApp("/admin/exams/new");
    const person = userEvent.setup();
    await screen.findByLabelText("Alpha?");
    screen.getByLabelText("Title").focus();

    await person.keyboard("Go{Tab}");
    expect(screen.getByLabelText("Open time")).toHaveFocus();
    await person.type(screen.getByLabelText("Open time"), "2026-11-01T09:00");
    await person.tab();
    expect(screen.getByLabelText("Close time")).toHaveFocus();
    await person.type(screen.getByLabelText("Close time"), "2026-11-01T17:00");
    await person.tab();
    expect(screen.getByLabelText("Time limit in minutes")).toHaveFocus();
    await person.tab();
    expect(screen.getByLabelText("Alpha?")).toHaveFocus();
    await person.keyboard(" ");
    await person.tab();
    await person.keyboard("{Enter}");

    await screen.findByText(/There is no Exam/);
    expect(received[0]?.questionIds).toEqual([a.id]);
  });
});

describe("the edit Exam page", () => {
  const a = question({ text: "Alpha?" });
  const b = question({ text: "Bravo?" });

  function examHandlers(existing: Exam) {
    return [
      http.get<{ id: string }, never, Exam>(examUrl, () => HttpResponse.json(existing)),
      questionListHandler([a, b]),
    ];
  }

  it("shows the Exam that it loads, with its Questions selected in order and its times in local time", async () => {
    loggedIn();
    const existing = exam({ title: "Go basics", questionIds: [b.id, a.id], timeLimitMinutes: 90 });
    server.use(...examHandlers(existing));
    renderApp(`/admin/exams/${existing.id}`);

    expect(await screen.findByLabelText("Title")).toHaveValue("Go basics");
    expect(screen.getByRole("heading", { level: 1, name: "Edit Exam" })).toBeInTheDocument();
    expect(screen.getByLabelText("Time limit in minutes")).toHaveValue(90);
    expect(screen.getByLabelText("Bravo?")).toBeChecked();
    expect(screen.getByLabelText("Alpha?")).toBeChecked();
    const items = within(screen.getByRole("list", { name: "Selected Questions" })).getAllByRole(
      "listitem",
    );
    expect(items.map((item) => item.textContent)).toEqual(["Bravo?", "Alpha?"]);
    const toInput = (iso: string) => {
      const date = new Date(iso);
      const pad = (n: number) => String(n).padStart(2, "0");
      return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
    };
    expect(screen.getByLabelText("Open time")).toHaveValue(toInput(existing.opensAt));
    expect(screen.getByLabelText("Close time")).toHaveValue(toInput(existing.closesAt));
  });

  it("sends a PUT with the changes and opens the list", async () => {
    loggedIn();
    const existing = exam({ title: "Old title", questionIds: [a.id] });
    const received: ExamRequest[] = [];
    server.use(
      ...examHandlers(existing),
      http.put<{ id: string }, ExamRequest, Exam>(examUrl, async ({ request, params }) => {
        expect(params.id).toBe(existing.id);
        received.push(await request.json());
        return HttpResponse.json(existing);
      }),
      examListHandler([examSummary({ title: "Old title" })]),
    );
    renderApp(`/admin/exams/${existing.id}`);
    const person = userEvent.setup();
    await screen.findByLabelText("Title");

    await person.clear(screen.getByLabelText("Title"));
    await person.type(screen.getByLabelText("Title"), "New title");
    await person.click(screen.getByLabelText("Bravo?"));
    await person.click(screen.getByRole("button", { name: "Save Exam" }));

    await screen.findByRole("row", { name: /Old title/ });
    expect(received).toHaveLength(1);
    expect(received[0]?.title).toBe("New title");
    expect(received[0]?.questionIds).toEqual([a.id, b.id]);
  });

  it("shows the detail of a 409 response when Candidates started the Exam", async () => {
    loggedIn();
    const existing = exam({ questionIds: [a.id] });
    const detail =
      "Candidates started this Exam. Only the close time can change, and you cannot delete the Exam.";
    server.use(
      ...examHandlers(existing),
      http.put(examUrl, () => HttpResponse.json(problem(409, { detail }), { status: 409 })),
    );
    renderApp(`/admin/exams/${existing.id}`);
    const person = userEvent.setup();
    await screen.findByLabelText("Title");

    await person.click(screen.getByRole("button", { name: "Save Exam" }));

    expect(await screen.findByText(detail)).toBeInTheDocument();
  });

  it("shows the Exam link, and copies it", async () => {
    loggedIn();
    const existing = exam({ questionIds: [a.id], linkToken: "abcdefghijklmnopqrstuv" });
    server.use(...examHandlers(existing));
    renderApp(`/admin/exams/${existing.id}`);
    const person = userEvent.setup();
    await screen.findByLabelText("Title");
    const link = `${window.location.origin}/e/abcdefghijklmnopqrstuv`;

    expect(screen.getByText(link)).toBeInTheDocument();
    await person.click(screen.getByRole("button", { name: "Copy link" }));

    expect(await screen.findByText("The link is copied.")).toBeInTheDocument();
    expect(await navigator.clipboard.readText()).toBe(link);
  });

  it("asks to confirm a regeneration, says that the old link stops working, and shows the new link", async () => {
    loggedIn();
    const existing = exam({ questionIds: [a.id], linkToken: "abcdefghijklmnopqrstuv" });
    const regenerated = { ...existing, linkToken: "ZYXWVUTSRQPONMLKJIHGFE" };
    let calls = 0;
    server.use(
      ...examHandlers(existing),
      http.post<{ id: string }, never, Exam>(examLinkUrl, ({ params }) => {
        expect(params.id).toBe(existing.id);
        calls += 1;
        return HttpResponse.json(regenerated);
      }),
    );
    renderApp(`/admin/exams/${existing.id}`);
    const person = userEvent.setup();
    await screen.findByLabelText("Title");

    await person.click(screen.getByRole("button", { name: "Regenerate link" }));
    expect(screen.getByText(/The old link stops working/)).toBeInTheDocument();
    expect(calls).toBe(0);
    await person.click(screen.getByRole("button", { name: "Confirm regenerate" }));

    expect(
      await screen.findByText(`${window.location.origin}/e/ZYXWVUTSRQPONMLKJIHGFE`),
    ).toBeInTheDocument();
    expect(screen.queryByText(/abcdefghijklmnopqrstuv/)).not.toBeInTheDocument();
    expect(calls).toBe(1);
  });

  it("does not regenerate when the Admin cancels", async () => {
    loggedIn();
    const existing = exam({ questionIds: [a.id] });
    server.use(...examHandlers(existing));
    renderApp(`/admin/exams/${existing.id}`);
    const person = userEvent.setup();
    await screen.findByLabelText("Title");

    await person.click(screen.getByRole("button", { name: "Regenerate link" }));
    await person.click(screen.getByRole("button", { name: "Cancel regenerate" }));

    expect(screen.getByRole("button", { name: "Regenerate link" })).toBeInTheDocument();
  });

  it("has a link to the results of the Exam", async () => {
    loggedIn();
    const existing = exam({ questionIds: [a.id] });
    server.use(...examHandlers(existing));
    renderApp(`/admin/exams/${existing.id}`);
    await screen.findByLabelText("Title");

    expect(screen.getByRole("link", { name: "Results" })).toHaveAttribute(
      "href",
      `/admin/exams/${existing.id}/results`,
    );
  });

  it("shows the detail of a 404 response for an Exam that does not exist, and no form", async () => {
    loggedIn();
    server.use(
      http.get(examUrl, () =>
        HttpResponse.json(problem(404, { detail: "The Exam does not exist. Check the ID." }), {
          status: 404,
        }),
      ),
      questionListHandler([]),
    );
    renderApp("/admin/exams/0198f2b0-0000-7000-8000-000000000000");

    expect(await screen.findByText("The Exam does not exist. Check the ID.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save Exam" })).not.toBeInTheDocument();
  });
});

describe("the results page", () => {
  const existing = exam({ title: "Go basics" });

  function resultsHandlers(rows: AttemptResult[]) {
    return [
      http.get<{ id: string }, never, Exam>(examUrl, () => HttpResponse.json(existing)),
      http.get<{ id: string }, never, AttemptResult[]>(examAttemptsUrl, () =>
        HttpResponse.json(rows),
      ),
    ];
  }

  it("shows a row for each Attempt with the name, the email address, the status, the Score and the times", async () => {
    loggedIn();
    const submitted = attemptResult({ name: "Bob", emailAddress: "bob@example.com" });
    const open = attemptResult({
      name: "Ann",
      emailAddress: "ann@example.com",
      status: "open",
      score: null,
      finishedAt: null,
    });
    const timedOut = attemptResult({
      name: "Cy",
      emailAddress: "cy@example.com",
      status: "timed_out",
      score: { correct: 0, total: 2 },
    });
    server.use(...resultsHandlers([submitted, open, timedOut]));
    renderApp(`/admin/exams/${existing.id}/results`);

    expect(
      await screen.findByRole("heading", { level: 1, name: "Results of Go basics" }),
    ).toBeInTheDocument();
    const bob = await screen.findByRole("row", { name: /Bob/ });
    expect(bob).toHaveTextContent("bob@example.com");
    expect(bob).toHaveTextContent("Submitted");
    expect(bob).toHaveTextContent("1 of 2");
    expect(bob).toHaveTextContent(local(submitted.startedAt));
    expect(bob).toHaveTextContent(local(submitted.finishedAt ?? ""));
    const ann = screen.getByRole("row", { name: /Ann/ });
    expect(ann).toHaveTextContent("Open");
    expect(ann).toHaveTextContent("Not available");
    expect(screen.getByRole("row", { name: /Cy/ })).toHaveTextContent("Timed out");
    expect(screen.getByRole("row", { name: /Cy/ })).toHaveTextContent("0 of 2");
    expect(document.title).toBe("Results - go-react-template");
  });

  it("keeps the order of the rows that the API gives", async () => {
    loggedIn();
    server.use(
      ...resultsHandlers([attemptResult({ name: "Zed" }), attemptResult({ name: "Abe" })]),
    );
    renderApp(`/admin/exams/${existing.id}/results`);

    await screen.findByRole("row", { name: /Zed/ });
    const names = screen
      .getAllByRole("row")
      .slice(1)
      .map((row) => row.textContent ?? "");
    expect(names[0]).toContain("Zed");
    expect(names[1]).toContain("Abe");
  });

  it("shows an empty state when no Candidate started the Exam", async () => {
    loggedIn();
    server.use(...resultsHandlers([]));
    renderApp(`/admin/exams/${existing.id}/results`);

    expect(await screen.findByText("No Candidate started this Exam.")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("shows the detail of a 404 response for an Exam that does not exist", async () => {
    loggedIn();
    server.use(
      http.get(examUrl, () =>
        HttpResponse.json(problem(404, { detail: "The Exam does not exist. Check the ID." }), {
          status: 404,
        }),
      ),
      http.get(examAttemptsUrl, () =>
        HttpResponse.json(problem(404, { detail: "The Exam does not exist. Check the ID." }), {
          status: 404,
        }),
      ),
    );
    renderApp("/admin/exams/0198f2b0-0000-7000-8000-000000000000/results");

    expect(
      (await screen.findAllByText("The Exam does not exist. Check the ID.")).length,
    ).toBeGreaterThan(0);
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });
});
