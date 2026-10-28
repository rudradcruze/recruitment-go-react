import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, http, HttpResponse } from "msw";
import { describe, expect, it } from "vite-plus/test";

import { admin, type Admin } from "../../fixtures/admin";
import { problem, type Problem } from "../../fixtures/problem";
import { question, type Question, type QuestionRequest } from "../../fixtures/question";
import { server } from "../../mocks/server";
import { questionUrl, questionsUrl, sessionUrl } from "../../mocks/urls";
import { renderApp } from "../../render";

// An Admin page needs a session, so each test starts with a session.
function loggedIn() {
  server.use(http.get<never, never, Admin>(sessionUrl, () => HttpResponse.json(admin())));
}

function listHandler(questions: Question[]) {
  return http.get<never, never, Question[]>(questionsUrl, () => HttpResponse.json(questions));
}

describe("the Question list page", () => {
  it("shows each Question with its number of Options, and a link to a new Question", async () => {
    loggedIn();
    const first = question({ text: "What is 2 + 2?" });
    server.use(listHandler([first]));
    renderApp("/admin/questions");

    expect(await screen.findByRole("heading", { level: 1, name: "Questions" })).toBeInTheDocument();
    const row = await screen.findByRole("row", { name: /What is 2 \+ 2\?/ });
    expect(within(row).getByText("4")).toBeInTheDocument();
    expect(within(row).getByRole("link", { name: "Edit" })).toHaveAttribute(
      "href",
      `/admin/questions/${first.id}`,
    );
    expect(screen.getByRole("link", { name: "New Question" })).toHaveAttribute(
      "href",
      "/admin/questions/new",
    );
    expect(document.title).toBe("Questions - go-react-template");
  });

  it("is reachable from the link in the Admin area", async () => {
    loggedIn();
    server.use(listHandler([]));
    renderApp("/admin");
    const person = userEvent.setup();

    await person.click(await screen.findByRole("link", { name: "Questions" }));

    expect(await screen.findByRole("heading", { level: 1, name: "Questions" })).toBeInTheDocument();
  });

  it("shows a message while it loads, and an empty state when the Question bank has no Question", async () => {
    loggedIn();
    server.use(
      http.get(questionsUrl, async () => {
        await delay(30);
        return HttpResponse.json([]);
      }),
    );
    renderApp("/admin/questions");

    expect(await screen.findByText("The application loads the Questions.")).toBeInTheDocument();
    expect(await screen.findByText(/The Question bank has no Questions/)).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("shows the message of the API when the list does not load", async () => {
    loggedIn();
    server.use(
      http.get(questionsUrl, () =>
        HttpResponse.json(
          problem(500, {
            detail: "The server cannot complete the request. Try again in 1 minute.",
          }),
          { status: 500 },
        ),
      ),
    );
    renderApp("/admin/questions");

    expect(
      await screen.findByText("The server cannot complete the request. Try again in 1 minute."),
    ).toBeInTheDocument();
  });

  it("asks to confirm a delete, deletes the Question, and reloads the list", async () => {
    loggedIn();
    const first = question({ text: "First?" });
    const second = question({ text: "Second?" });
    let remaining = [first, second];
    const deleted: string[] = [];
    server.use(
      http.get<never, never, Question[]>(questionsUrl, () => HttpResponse.json(remaining)),
      http.delete<{ id: string }>(questionUrl, ({ params }) => {
        deleted.push(params.id);
        remaining = remaining.filter((q) => q.id !== params.id);
        return new HttpResponse(null, { status: 204 });
      }),
    );
    renderApp("/admin/questions");
    const person = userEvent.setup();
    const row = await screen.findByRole("row", { name: /First\?/ });

    await person.click(within(row).getByRole("button", { name: "Delete" }));
    expect(deleted).toEqual([]);
    await person.click(within(row).getByRole("button", { name: "Confirm delete" }));

    expect(await screen.findByText("The Question is deleted.")).toBeInTheDocument();
    await screen.findByRole("row", { name: /Second\?/ });
    expect(screen.queryByRole("row", { name: /First\?/ })).not.toBeInTheDocument();
    expect(deleted).toEqual([first.id]);
  });

  it("does not delete when the person cancels", async () => {
    loggedIn();
    server.use(listHandler([question({ text: "First?" })]));
    renderApp("/admin/questions");
    const person = userEvent.setup();
    const row = await screen.findByRole("row", { name: /First\?/ });

    await person.click(within(row).getByRole("button", { name: "Delete" }));
    await person.click(within(row).getByRole("button", { name: "Cancel" }));

    expect(within(row).getByRole("button", { name: "Delete" })).toBeInTheDocument();
    expect(within(row).queryByRole("button", { name: "Confirm delete" })).not.toBeInTheDocument();
  });

  it("shows the detail of a 409 response when an Exam uses the Question", async () => {
    loggedIn();
    const detail =
      "An Exam uses the Question. Remove the Question from the Exam, or delete the Exam. Then delete the Question.";
    server.use(
      listHandler([question({ text: "First?" })]),
      http.delete(questionUrl, () => HttpResponse.json(problem(409, { detail }), { status: 409 })),
    );
    renderApp("/admin/questions");
    const person = userEvent.setup();
    const row = await screen.findByRole("row", { name: /First\?/ });

    await person.click(within(row).getByRole("button", { name: "Delete" }));
    await person.click(within(row).getByRole("button", { name: "Confirm delete" }));

    expect(await screen.findByText(detail)).toBeInTheDocument();
    expect(screen.getByRole("row", { name: /First\?/ })).toBeInTheDocument();
  });
});

// listLoaded waits until the Question list page has its empty list. The page requests the list after a save,
// and a request that arrives after the test ends fails the next test.
async function listLoaded() {
  await screen.findByRole("heading", { level: 1, name: "Questions" });
  await screen.findByText(/The Question bank has no Questions/);
}

async function fillOption(person: ReturnType<typeof userEvent.setup>, n: number, text: string) {
  await person.type(screen.getByLabelText(`Option ${n}`), text);
}

describe("the new Question page", () => {
  it("has a label for each field, 2 Options, and the region of the error message", () => {
    loggedIn();
    renderApp("/admin/questions/new");

    return screen.findByRole("heading", { level: 1, name: "New Question" }).then(() => {
      expect(screen.getByLabelText("Question text")).toBeInTheDocument();
      expect(screen.getByLabelText("Option 1")).toBeInTheDocument();
      expect(screen.getByLabelText("Option 2")).toBeInTheDocument();
      expect(screen.queryByLabelText("Option 3")).not.toBeInTheDocument();
      expect(screen.getByLabelText("Option 1 is correct")).not.toBeChecked();
      expect(screen.getByRole("button", { name: "Save Question" })).toBeInTheDocument();
      expect(screen.getAllByRole("alert").some((region) => region.textContent === "")).toBe(true);
      expect(document.title).toBe("New Question - go-react-template");
    });
  });

  it("sends the text, the Options and the correct Option, and opens the list", async () => {
    loggedIn();
    const received: QuestionRequest[] = [];
    server.use(
      http.post<never, QuestionRequest, Question>(questionsUrl, async ({ request }) => {
        received.push(await request.json());
        return HttpResponse.json(question(), { status: 201 });
      }),
      listHandler([]),
    );
    const { router } = renderApp("/admin/questions/new");
    const person = userEvent.setup();
    await screen.findByRole("heading", { level: 1, name: "New Question" });

    await person.type(screen.getByLabelText("Question text"), "What is 2 + 2?");
    await fillOption(person, 1, "3");
    await fillOption(person, 2, "4");
    await person.click(screen.getByLabelText("Option 2 is correct"));
    await person.click(screen.getByRole("button", { name: "Save Question" }));

    await listLoaded();
    expect(received).toEqual([
      {
        text: "What is 2 + 2?",
        options: [
          { text: "3", correct: false },
          { text: "4", correct: true },
        ],
      },
    ]);
    expect(router.state.location.pathname).toBe("/admin/questions");
    expect(router.state.historyAction).toBe("REPLACE");
  });

  it("adds Options up to 6 and removes Options down to 2", async () => {
    loggedIn();
    renderApp("/admin/questions/new");
    const person = userEvent.setup();
    await screen.findByRole("heading", { level: 1, name: "New Question" });

    for (let n = 3; n <= 6; n += 1) {
      await person.click(screen.getByRole("button", { name: "Add Option" }));
      expect(screen.getByLabelText(`Option ${n}`)).toBeInTheDocument();
    }
    await person.click(screen.getByRole("button", { name: "Add Option" }));
    expect(screen.queryByLabelText("Option 7")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add Option" })).toHaveAttribute(
      "aria-disabled",
      "true",
    );

    for (let n = 6; n >= 3; n -= 1) {
      await person.click(screen.getByRole("button", { name: `Remove Option ${n}` }));
    }
    expect(screen.queryByLabelText("Option 3")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Remove Option/ })).not.toBeInTheDocument();
  });

  it("keeps the correct mark on the same Option when the person removes an Option before it", async () => {
    loggedIn();
    const received: QuestionRequest[] = [];
    server.use(
      http.post<never, QuestionRequest, Question>(questionsUrl, async ({ request }) => {
        received.push(await request.json());
        return HttpResponse.json(question(), { status: 201 });
      }),
      listHandler([]),
    );
    renderApp("/admin/questions/new");
    const person = userEvent.setup();
    await screen.findByRole("heading", { level: 1, name: "New Question" });
    await person.click(screen.getByRole("button", { name: "Add Option" }));
    await person.type(screen.getByLabelText("Question text"), "Q");
    await fillOption(person, 1, "one");
    await fillOption(person, 2, "two");
    await fillOption(person, 3, "three");
    await person.click(screen.getByLabelText("Option 3 is correct"));

    await person.click(screen.getByRole("button", { name: "Remove Option 1" }));
    await person.click(screen.getByRole("button", { name: "Save Question" }));

    await listLoaded();
    expect(received[0]?.options).toEqual([
      { text: "two", correct: false },
      { text: "three", correct: true },
    ]);
  });

  it("shows the detail of a 422 response and stays on the form with the values", async () => {
    loggedIn();
    const detail = "The Question must have exactly one correct Option. Mark one Option as correct.";
    server.use(
      http.post(questionsUrl, () => HttpResponse.json(problem(422, { detail }), { status: 422 })),
    );
    const { router } = renderApp("/admin/questions/new");
    const person = userEvent.setup();
    await screen.findByRole("heading", { level: 1, name: "New Question" });
    await person.type(screen.getByLabelText("Question text"), "Q");

    await person.click(screen.getByRole("button", { name: "Save Question" }));

    expect(await screen.findByText(detail)).toBeInTheDocument();
    expect(screen.getByLabelText("Question text")).toHaveValue("Q");
    expect(router.state.location.pathname).toBe("/admin/questions/new");
  });

  it("does not send the form twice while it waits for the server", async () => {
    loggedIn();
    let requests = 0;
    server.use(
      http.post(questionsUrl, async () => {
        requests += 1;
        await delay(100);
        return HttpResponse.json(question(), { status: 201 });
      }),
      listHandler([]),
    );
    renderApp("/admin/questions/new");
    const person = userEvent.setup();
    await screen.findByRole("heading", { level: 1, name: "New Question" });

    await person.click(screen.getByRole("button", { name: "Save Question" }));
    const button = screen.getByRole("button", { name: "Save Question" });
    expect(button).toHaveAttribute("aria-disabled", "true");
    expect(button).toHaveFocus();
    await person.click(button);

    await listLoaded();
    expect(requests).toBe(1);
  });

  it("works with the keyboard only", async () => {
    loggedIn();
    const received: QuestionRequest[] = [];
    server.use(
      http.post<never, QuestionRequest, Question>(questionsUrl, async ({ request }) => {
        received.push(await request.json());
        return HttpResponse.json(question(), { status: 201 });
      }),
      listHandler([]),
    );
    renderApp("/admin/questions/new");
    const person = userEvent.setup();
    await screen.findByRole("heading", { level: 1, name: "New Question" });
    screen.getByLabelText("Question text").focus();

    await person.keyboard("Q{Tab}one{Tab}");
    expect(screen.getByLabelText("Option 1 is correct")).toHaveFocus();
    await person.keyboard(" {Tab}");
    expect(screen.getByLabelText("Option 2")).toHaveFocus();
    await person.keyboard("two{Enter}");

    await listLoaded();
    expect(received).toEqual([
      {
        text: "Q",
        options: [
          { text: "one", correct: true },
          { text: "two", correct: false },
        ],
      },
    ]);
  });
});

describe("the edit Question page", () => {
  it("shows the Question that it loads, with the correct Option marked", async () => {
    loggedIn();
    const existing = question({ text: "What is 2 + 2?" });
    server.use(
      http.get<{ id: string }, never, Question>(questionUrl, () => HttpResponse.json(existing)),
    );
    renderApp(`/admin/questions/${existing.id}`);

    expect(await screen.findByLabelText("Question text")).toHaveValue("What is 2 + 2?");
    expect(screen.getByRole("heading", { level: 1, name: "Edit Question" })).toBeInTheDocument();
    expect(screen.getByLabelText("Option 1")).toHaveValue(existing.options[0]?.text);
    expect(screen.getByLabelText("Option 4")).toHaveValue(existing.options[3]?.text);
    expect(screen.getByLabelText("Option 1 is correct")).toBeChecked();
    expect(screen.getByLabelText("Option 2 is correct")).not.toBeChecked();
  });

  it("sends a PUT with the changes and opens the list", async () => {
    loggedIn();
    const existing = question({ text: "Old text" });
    const received: QuestionRequest[] = [];
    server.use(
      http.get<{ id: string }, never, Question>(questionUrl, () => HttpResponse.json(existing)),
      http.put<{ id: string }, QuestionRequest, Question>(
        questionUrl,
        async ({ request, params }) => {
          expect(params.id).toBe(existing.id);
          received.push(await request.json());
          return HttpResponse.json(existing);
        },
      ),
      listHandler([existing]),
    );
    renderApp(`/admin/questions/${existing.id}`);
    const person = userEvent.setup();
    await screen.findByLabelText("Question text");

    await person.clear(screen.getByLabelText("Question text"));
    await person.type(screen.getByLabelText("Question text"), "New text");
    await person.click(screen.getByLabelText("Option 3 is correct"));
    await person.click(screen.getByRole("button", { name: "Save Question" }));

    await screen.findByRole("row", { name: /Old text/ });
    expect(received).toHaveLength(1);
    expect(received[0]?.text).toBe("New text");
    expect(received[0]?.options.map((o) => o.correct)).toEqual([false, false, true, false]);
  });

  it("shows the detail of a 409 response when an Attempt used the Question", async () => {
    loggedIn();
    const existing = question();
    const detail = "An Attempt used this Question. You cannot change the Question.";
    server.use(
      http.get<{ id: string }, never, Question>(questionUrl, () => HttpResponse.json(existing)),
      http.put(questionUrl, () => HttpResponse.json(problem(409, { detail }), { status: 409 })),
    );
    renderApp(`/admin/questions/${existing.id}`);
    const person = userEvent.setup();
    await screen.findByLabelText("Question text");

    await person.click(screen.getByRole("button", { name: "Save Question" }));

    expect(await screen.findByText(detail)).toBeInTheDocument();
  });

  it("shows the detail of a 404 response for a Question that does not exist, and no form", async () => {
    loggedIn();
    server.use(
      http.get<{ id: string }, never, Problem>(questionUrl, () =>
        HttpResponse.json(problem(404, { detail: "The Question does not exist. Check the ID." }), {
          status: 404,
        }),
      ),
    );
    renderApp("/admin/questions/0198f2a0-0000-7000-8000-000000000000");

    expect(
      await screen.findByText("The Question does not exist. Check the ID."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save Question" })).not.toBeInTheDocument();
  });
});
