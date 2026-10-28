import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, http, HttpResponse } from "msw";
import { describe, expect, it } from "vite-plus/test";

import { attempt, type Attempt, type SaveAnswerRequest, type Score } from "../../fixtures/attempt";
import { problem } from "../../fixtures/problem";
import { server } from "../../mocks/server";
import { answerUrl, attemptUrl, submissionUrl } from "../../mocks/urls";
import { renderApp } from "../../render";

function attemptHandler(view: Attempt) {
  return http.get<{ id: string }, never, Attempt>(attemptUrl, () => HttpResponse.json(view));
}

function submissionHandler(score: Score) {
  return http.post<{ id: string }, never, Score>(submissionUrl, () => HttpResponse.json(score));
}

const finishedMessage = "This Exam is already submitted.";

describe("the Attempt page", () => {
  it("shows the title of the Exam, a countdown, and all Questions with their Options as radio buttons in a group", async () => {
    const view = attempt({ examTitle: "Go basics" });
    server.use(attemptHandler(view));
    renderApp(`/attempts/${view.id}`);

    expect(await screen.findByRole("heading", { level: 1, name: "Go basics" })).toBeInTheDocument();
    expect(screen.getByRole("timer")).toHaveTextContent(/^(29|30):\d{2}$/);
    for (const question of view.questions ?? []) {
      const group = screen.getByRole("group", { name: question.text });
      expect(within(group).getAllByRole("radio")).toHaveLength(3);
      for (const option of question.options) {
        expect(within(group).getByLabelText(option.text)).not.toBeChecked();
      }
    }
    expect(screen.getByRole("button", { name: "Submit" })).toBeInTheDocument();
    expect(document.title).toBe("Go basics - go-react-template");
  });

  it("shows a message while it loads, and the message of the API when the Attempt does not exist", async () => {
    server.use(
      http.get(attemptUrl, async () => {
        await delay(30);
        return HttpResponse.json(
          problem(404, { detail: "The Attempt does not exist. Check the address of the page." }),
          { status: 404 },
        );
      }),
    );
    renderApp("/attempts/0198f2c0-0000-4000-8000-000000000000");

    expect(screen.getByRole("status")).toHaveTextContent("The application loads the Exam.");
    expect(
      await screen.findByText("The Attempt does not exist. Check the address of the page."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Submit" })).not.toBeInTheDocument();
  });

  it("shows the saved Answers again after a refresh", async () => {
    const view = attempt();
    const [first, second] = view.questions ?? [];
    view.answers = [
      { questionId: first?.id ?? "", optionId: first?.options[2]?.id ?? "" },
      { questionId: second?.id ?? "", optionId: second?.options[0]?.id ?? "" },
    ];
    server.use(attemptHandler(view));
    renderApp(`/attempts/${view.id}`);

    expect(await screen.findByLabelText(first?.options[2]?.text ?? "")).toBeChecked();
    expect(screen.getByLabelText(second?.options[0]?.text ?? "")).toBeChecked();
    expect(screen.getByLabelText(first?.options[0]?.text ?? "")).not.toBeChecked();
  });

  it("sends an Answer as soon as the Candidate selects an Option, and says that it is saved", async () => {
    const view = attempt();
    const question = view.questions?.[0];
    const received: Array<{ attemptId: string; questionId: string; body: SaveAnswerRequest }> = [];
    server.use(
      attemptHandler(view),
      http.put<{ id: string; questionId: string }, SaveAnswerRequest>(
        answerUrl,
        async ({ request, params }) => {
          received.push({
            attemptId: params.id,
            questionId: params.questionId,
            body: await request.json(),
          });
          return new HttpResponse(null, { status: 204 });
        },
      ),
    );
    renderApp(`/attempts/${view.id}`);
    const person = userEvent.setup();

    await person.click(await screen.findByLabelText(question?.options[1]?.text ?? ""));

    expect(await screen.findByText("The Answer is saved.")).toBeInTheDocument();
    expect(received).toEqual([
      {
        attemptId: view.id,
        questionId: question?.id,
        body: { optionId: question?.options[1]?.id },
      },
    ]);
    expect(screen.getByLabelText(question?.options[1]?.text ?? "")).toBeChecked();
  });

  it("sends the new Option when the Candidate changes an Answer", async () => {
    const view = attempt();
    const question = view.questions?.[0];
    const sent: string[] = [];
    server.use(
      attemptHandler(view),
      http.put<{ id: string; questionId: string }, SaveAnswerRequest>(
        answerUrl,
        async ({ request }) => {
          sent.push((await request.json()).optionId);
          return new HttpResponse(null, { status: 204 });
        },
      ),
    );
    renderApp(`/attempts/${view.id}`);
    const person = userEvent.setup();

    await person.click(await screen.findByLabelText(question?.options[0]?.text ?? ""));
    await screen.findByText("The Answer is saved.");
    await person.click(screen.getByLabelText(question?.options[2]?.text ?? ""));

    await screen.findByLabelText(question?.options[2]?.text ?? "");
    expect(sent).toEqual([question?.options[0]?.id, question?.options[2]?.id]);
    expect(screen.getByLabelText(question?.options[2]?.text ?? "")).toBeChecked();
    expect(screen.getByLabelText(question?.options[0]?.text ?? "")).not.toBeChecked();
  });

  it("shows the detail of a failed save and returns the selection to the saved Answer", async () => {
    const view = attempt();
    const question = view.questions?.[0];
    const detail =
      "The Attempt is closed. The Time limit ended, or you submitted the Attempt. You cannot save more Answers.";
    server.use(
      attemptHandler(view),
      http.put(answerUrl, () => HttpResponse.json(problem(409, { detail }), { status: 409 })),
    );
    renderApp(`/attempts/${view.id}`);
    const person = userEvent.setup();

    await person.click(await screen.findByLabelText(question?.options[1]?.text ?? ""));

    expect(await screen.findByText(detail)).toBeInTheDocument();
    expect(screen.getByLabelText(question?.options[1]?.text ?? "")).not.toBeChecked();
  });

  it("submits, and shows the Score and no Questions", async () => {
    const view = attempt();
    server.use(attemptHandler(view), submissionHandler({ correct: 7, total: 10 }));
    renderApp(`/attempts/${view.id}`);
    const person = userEvent.setup();

    await person.click(await screen.findByRole("button", { name: "Submit" }));

    expect(await screen.findByText("Your Score is 7 of 10.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Submit" })).not.toBeInTheDocument();
    expect(screen.queryByRole("group")).not.toBeInTheDocument();
    expect(screen.queryByRole("timer")).not.toBeInTheDocument();
  });

  it("submits by itself when the Time limit is over, and shows the Score", async () => {
    const view = attempt({ expiresAt: new Date(Date.now() - 1000).toISOString() });
    let submissions = 0;
    server.use(
      attemptHandler(view),
      http.post(submissionUrl, () => {
        submissions += 1;
        return HttpResponse.json({ correct: 1, total: 2 });
      }),
    );
    renderApp(`/attempts/${view.id}`);

    expect(await screen.findByText("Your Score is 1 of 2.")).toBeInTheDocument();
    expect(submissions).toBe(1);
  });

  it("submits by itself when the countdown reaches 0", async () => {
    const view = attempt({ expiresAt: new Date(Date.now() + 1500).toISOString() });
    server.use(attemptHandler(view), submissionHandler({ correct: 0, total: 2 }));
    renderApp(`/attempts/${view.id}`);

    expect(await screen.findByRole("timer")).toBeInTheDocument();
    expect(
      await screen.findByText("Your Score is 0 of 2.", undefined, { timeout: 4000 }),
    ).toBeInTheDocument();
  });

  it("does not send the Submit twice while it waits for the server, and the button keeps the focus", async () => {
    const view = attempt();
    let submissions = 0;
    server.use(
      attemptHandler(view),
      http.post(submissionUrl, async () => {
        submissions += 1;
        await delay(100);
        return HttpResponse.json({ correct: 1, total: 2 });
      }),
    );
    renderApp(`/attempts/${view.id}`);
    const person = userEvent.setup();

    await person.click(await screen.findByRole("button", { name: "Submit" }));
    const button = screen.getByRole("button", { name: "Submit" });
    expect(button).toHaveAttribute("aria-disabled", "true");
    expect(button).toHaveFocus();
    await person.click(button);

    await screen.findByText("Your Score is 1 of 2.");
    expect(submissions).toBe(1);
  });

  it("shows the detail of a failed Submit and keeps the Questions", async () => {
    const view = attempt();
    const detail = "The Attempt is already submitted. You cannot submit it again.";
    server.use(
      attemptHandler(view),
      http.post(submissionUrl, () => HttpResponse.json(problem(409, { detail }), { status: 409 })),
    );
    renderApp(`/attempts/${view.id}`);
    const person = userEvent.setup();

    await person.click(await screen.findByRole("button", { name: "Submit" }));

    expect(await screen.findByText(detail)).toBeInTheDocument();
    expect(
      screen.getByRole("group", { name: view.questions?.[0]?.text ?? "" }),
    ).toBeInTheDocument();
  });

  it.each([["submitted"], ["timed_out"]] as const)(
    "shows only a message for an Attempt with the status %s, and no Score and no Questions",
    async (status) => {
      const view: Attempt = { id: "0198f2c0-0000-4000-8000-0000000000aa", status };
      server.use(attemptHandler(view));
      renderApp(`/attempts/${view.id}`);

      expect(await screen.findByText(finishedMessage)).toBeInTheDocument();
      expect(screen.queryByText(/Score/)).not.toBeInTheDocument();
      expect(screen.queryByRole("group")).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Submit" })).not.toBeInTheDocument();
    },
  );

  it("works with the keyboard only", async () => {
    const view = attempt();
    const question = view.questions?.[0];
    const sent: string[] = [];
    server.use(
      attemptHandler(view),
      http.put<{ id: string; questionId: string }, SaveAnswerRequest>(
        answerUrl,
        async ({ request }) => {
          sent.push((await request.json()).optionId);
          return new HttpResponse(null, { status: 204 });
        },
      ),
      submissionHandler({ correct: 1, total: 2 }),
    );
    renderApp(`/attempts/${view.id}`);
    const person = userEvent.setup();
    await screen.findByRole("heading", { level: 1 });

    await person.tab();
    expect(screen.getByLabelText(question?.options[0]?.text ?? "")).toHaveFocus();
    await person.keyboard(" ");
    await screen.findByText("The Answer is saved.");
    expect(sent).toEqual([question?.options[0]?.id]);
    screen.getByRole("button", { name: "Submit" }).focus();
    await person.keyboard("{Enter}");

    expect(await screen.findByText("Your Score is 1 of 2.")).toBeInTheDocument();
  });
});
