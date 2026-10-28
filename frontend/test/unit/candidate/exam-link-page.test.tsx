import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, http, HttpResponse } from "msw";
import { describe, expect, it } from "vite-plus/test";

import {
  attempt,
  examLink,
  type Attempt,
  type ExamLink,
  type StartAttemptRequest,
} from "../../fixtures/attempt";
import { problem } from "../../fixtures/problem";
import { server } from "../../mocks/server";
import { examLinkInfoUrl, startAttemptUrl } from "../../mocks/urls";
import { renderApp } from "../../render";

const token = "abcdefghijklmnopqrstuv";

function infoHandler(info: ExamLink) {
  return http.get<{ token: string }, never, ExamLink>(examLinkInfoUrl, () =>
    HttpResponse.json(info),
  );
}

async function fillAndSubmit(request: StartAttemptRequest) {
  const person = userEvent.setup();
  await person.type(screen.getByLabelText("Name"), request.name);
  await person.type(screen.getByLabelText("Email address"), request.emailAddress);
  await person.click(screen.getByRole("button", { name: "Start" }));
}

describe("the Exam link page", () => {
  it("shows the title, the Exam window in local time and the Time limit, and a form with a label for each field", async () => {
    const info = examLink({ title: "Go basics", timeLimitMinutes: 45 });
    server.use(infoHandler(info));
    renderApp(`/e/${token}`);

    expect(await screen.findByRole("heading", { level: 1, name: "Go basics" })).toBeInTheDocument();
    expect(
      screen.getByText(new Date(info.opensAt).toLocaleString(), { exact: false }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(new Date(info.closesAt).toLocaleString(), { exact: false }),
    ).toBeInTheDocument();
    expect(screen.getByText(/45 minutes/)).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toBeInTheDocument();
    expect(screen.getByLabelText("Email address")).toHaveAttribute("type", "email");
    expect(screen.getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(screen.getByRole("alert")).toBeEmptyDOMElement();
    expect(document.title).toBe("Go basics - go-react-template");
  });

  it("shows a message while it loads", async () => {
    server.use(
      http.get(examLinkInfoUrl, async () => {
        await delay(30);
        return HttpResponse.json(examLink());
      }),
    );
    renderApp(`/e/${token}`);

    expect(screen.getByRole("status")).toHaveTextContent("The application loads the Exam.");
    await screen.findByRole("button", { name: "Start" });
  });

  it("shows the message of the API and no form when the Exam link is not valid", async () => {
    server.use(
      http.get(examLinkInfoUrl, () =>
        HttpResponse.json(problem(404, { detail: "The Exam link is not valid. Check the link." }), {
          status: 404,
        }),
      ),
    );
    renderApp(`/e/${token}`);

    expect(
      await screen.findByText("The Exam link is not valid. Check the link."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Start" })).not.toBeInTheDocument();
  });

  it("sends the name and the email address, and opens the Attempt page with the Questions of the response", async () => {
    // The test has no handler for the Attempt page. A request for the Attempt would fail the test.
    const received: StartAttemptRequest[] = [];
    const started = attempt({ examTitle: "Go basics" });
    server.use(
      infoHandler(examLink({ title: "Go basics" })),
      http.post<{ token: string }, StartAttemptRequest, Attempt>(
        startAttemptUrl,
        async ({ request, params }) => {
          expect(params.token).toBe(token);
          received.push(await request.json());
          return HttpResponse.json(started, { status: 201 });
        },
      ),
    );
    const { router } = renderApp(`/e/${token}`);
    await screen.findByRole("button", { name: "Start" });

    await fillAndSubmit({ name: "Ann Smith", emailAddress: "ann@example.com" });

    expect(await screen.findByText(started.questions?.[0]?.text ?? "")).toBeInTheDocument();
    expect(received).toEqual([{ name: "Ann Smith", emailAddress: "ann@example.com" }]);
    expect(router.state.location.pathname).toBe(`/attempts/${started.id}`);
    expect(router.state.historyAction).toBe("REPLACE");
  });

  it("opens the open Attempt that the API gives for a start with an email address that has one (status 200)", async () => {
    const resumed = attempt();
    server.use(
      infoHandler(examLink()),
      http.post(startAttemptUrl, () => HttpResponse.json(resumed, { status: 200 })),
    );
    const { router } = renderApp(`/e/${token}`);
    await screen.findByRole("button", { name: "Start" });

    await fillAndSubmit({ name: "Ann", emailAddress: "ann@example.com" });

    await screen.findByText(resumed.questions?.[0]?.text ?? "");
    expect(router.state.location.pathname).toBe(`/attempts/${resumed.id}`);
  });

  it.each([
    [409, "The Exam is not open yet. Try again after the open time."],
    [409, "The Exam is closed. You cannot start it now."],
    [409, "This Exam is already submitted for this email address. You cannot start it again."],
    [422, "The email address is not valid. Enter an address in the format name@example.com."],
    [404, "The Exam link is not valid. Check the link."],
  ])(
    "shows the detail of a problem with the status %i and stays on the page",
    async (status, detail) => {
      server.use(
        infoHandler(examLink()),
        http.post(startAttemptUrl, () =>
          HttpResponse.json(problem(status, { detail }), { status }),
        ),
      );
      const { router } = renderApp(`/e/${token}`);
      await screen.findByRole("button", { name: "Start" });

      await fillAndSubmit({ name: "Ann", emailAddress: "ann@example.com" });

      expect(await screen.findByText(detail)).toBeInTheDocument();
      expect(screen.getByRole("alert")).toHaveTextContent(detail);
      expect(router.state.location.pathname).toBe(`/e/${token}`);
      expect(screen.getByLabelText("Name")).toHaveValue("Ann");
    },
  );

  it("shows a message when the server does not answer", async () => {
    server.use(
      infoHandler(examLink()),
      http.post(startAttemptUrl, () => HttpResponse.error()),
    );
    renderApp(`/e/${token}`);
    await screen.findByRole("button", { name: "Start" });

    await fillAndSubmit({ name: "Ann", emailAddress: "ann@example.com" });

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The application cannot reach the server.",
    );
  });

  it("does not send the form twice while it waits for the server, and the button keeps the focus", async () => {
    let requests = 0;
    const started = attempt();
    server.use(
      infoHandler(examLink()),
      http.post(startAttemptUrl, async () => {
        requests += 1;
        await delay(100);
        return HttpResponse.json(started, { status: 201 });
      }),
    );
    renderApp(`/e/${token}`);
    await screen.findByRole("button", { name: "Start" });
    const person = userEvent.setup();
    await person.type(screen.getByLabelText("Name"), "Ann");
    await person.type(screen.getByLabelText("Email address"), "ann@example.com");

    await person.click(screen.getByRole("button", { name: "Start" }));
    const button = screen.getByRole("button", { name: "Start" });
    expect(button).toHaveAttribute("aria-disabled", "true");
    expect(button).toHaveFocus();
    await person.click(button);

    await screen.findByText(started.questions?.[0]?.text ?? "");
    expect(requests).toBe(1);
  });

  it("works with the keyboard only", async () => {
    const received: StartAttemptRequest[] = [];
    const started = attempt();
    server.use(
      infoHandler(examLink()),
      http.post<never, StartAttemptRequest, Attempt>(startAttemptUrl, async ({ request }) => {
        received.push(await request.json());
        return HttpResponse.json(started, { status: 201 });
      }),
    );
    renderApp(`/e/${token}`);
    await screen.findByRole("button", { name: "Start" });
    const person = userEvent.setup();

    await person.tab();
    expect(screen.getByLabelText("Name")).toHaveFocus();
    await person.keyboard("Ann");
    await person.tab();
    expect(screen.getByLabelText("Email address")).toHaveFocus();
    await person.keyboard("ann@example.com{Enter}");

    await screen.findByText(started.questions?.[0]?.text ?? "");
    expect(received).toEqual([{ name: "Ann", emailAddress: "ann@example.com" }]);
  });
});
