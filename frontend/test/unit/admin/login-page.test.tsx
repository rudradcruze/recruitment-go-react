import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, http, HttpResponse } from "msw";
import { afterEach, beforeEach, describe, expect, it, vi } from "vite-plus/test";

import { admin, loginRequest, type Admin, type LoginRequest } from "../../fixtures/admin";
import { problem, type Problem } from "../../fixtures/problem";
import { server } from "../../mocks/server";
import { sessionUrl } from "../../mocks/urls";
import { renderApp } from "../../render";

const serverProblem = "The server cannot complete the request. Try again in 1 minute.";

async function fillAndSubmit(request: LoginRequest) {
  const person = userEvent.setup();
  await person.type(screen.getByLabelText("Email address"), request.emailAddress);
  await person.type(screen.getByLabelText("Password"), request.password);
  await person.click(screen.getByRole("button", { name: "Log in" }));
}

// loginHandlers answer the login with 204, and the session with the Admin. The Admin page reads the session.
function loginHandlers(received: LoginRequest[], ann: Admin = admin()) {
  return [
    http.post<never, LoginRequest>(sessionUrl, async ({ request }) => {
      received.push(await request.json());
      return new HttpResponse(null, { status: 204 });
    }),
    http.get<never, never, Admin>(sessionUrl, () => HttpResponse.json(ann)),
  ];
}

describe("the login page", () => {
  it("has a label for each field and a button", () => {
    renderApp("/admin/login");

    expect(screen.getByRole("heading", { level: 1, name: "Log in" })).toBeInTheDocument();
    expect(screen.getByLabelText("Email address")).toHaveAttribute("type", "email");
    expect(screen.getByLabelText("Password")).toHaveAttribute("type", "password");
    expect(screen.getByRole("button", { name: "Log in" })).toBeInTheDocument();
  });

  it("has the region of the error message before it has a message, so that a screen reader announces the first message", () => {
    renderApp("/admin/login");

    expect(screen.getByRole("alert")).toBeEmptyDOMElement();
  });

  it("has a page title, and tells the browser which fields hold the user name and the password", () => {
    renderApp("/admin/login");

    expect(document.title).toBe("Log in - go-react-template");
    expect(screen.getByLabelText("Email address")).toHaveAttribute("autocomplete", "username");
    expect(screen.getByLabelText("Password")).toHaveAttribute("autocomplete", "current-password");
  });

  it("sends the email address and the password that the person entered", async () => {
    const received: LoginRequest[] = [];
    server.use(...loginHandlers(received));
    renderApp("/admin/login");
    const request = loginRequest({
      emailAddress: "ann@example.com",
      password: "a password of 12+",
    });

    await fillAndSubmit(request);

    await screen.findByRole("heading", { level: 1, name: "Admin" });
    expect(received).toEqual([request]);
  });

  it("opens the Admin page after a login, with the email address of the Admin and a Log out button", async () => {
    server.use(...loginHandlers([], admin({ emailAddress: "ann@example.com" })));
    const { router } = renderApp("/admin/login");

    await fillAndSubmit(loginRequest());

    expect(await screen.findByText("ann@example.com")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Log out" })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe("/admin");
  });

  it("replaces the login page in the history, so the Back button does not return to the form", async () => {
    server.use(...loginHandlers([]));
    const { router } = renderApp("/admin/login");

    await fillAndSubmit(loginRequest());

    await screen.findByRole("heading", { level: 1, name: "Admin" });
    expect(router.state.historyAction).toBe("REPLACE");
  });

  it.each([
    [401, "The email address or the password is not correct. Enter them again."],
    [429, "There were too many failed logins for this email address. Wait, then log in again."],
    [500, serverProblem],
  ])(
    "shows the detail of a problem with the status %i and stays on the login page",
    async (status, detail) => {
      server.use(
        http.post<never, LoginRequest, Problem>(sessionUrl, () =>
          HttpResponse.json(problem(status, { detail }), { status }),
        ),
      );
      const { router } = renderApp("/admin/login");

      await fillAndSubmit(loginRequest());

      expect(await screen.findByText(detail)).toBeInTheDocument();
      expect(screen.getByRole("alert")).toHaveTextContent(detail);
      expect(router.state.location.pathname).toBe("/admin/login");
    },
  );

  it.each([
    ["an empty body from a proxy", () => new HttpResponse(null, { status: 502 })],
    ["an HTML body from a proxy", () => HttpResponse.html("<h1>Bad gateway</h1>", { status: 502 })],
  ])("shows a general message for a failed response with %s", async (_name, respond) => {
    server.use(http.post(sessionUrl, respond));
    renderApp("/admin/login");

    await fillAndSubmit(loginRequest());

    expect(await screen.findByText(serverProblem)).toBeInTheDocument();
  });

  it("shows a message when the server does not answer", async () => {
    server.use(http.post(sessionUrl, () => HttpResponse.error()));
    renderApp("/admin/login");

    await fillAndSubmit(loginRequest());

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The application cannot reach the server.",
    );
  });

  it("does not send the form twice while it waits for the server, and the button keeps the focus", async () => {
    let requests = 0;
    server.use(
      http.post(sessionUrl, async () => {
        requests += 1;
        await delay(100);
        return new HttpResponse(null, { status: 204 });
      }),
      http.get<never, never, Admin>(sessionUrl, () => HttpResponse.json(admin())),
    );
    renderApp("/admin/login");
    const person = userEvent.setup();
    const request = loginRequest();
    await person.type(screen.getByLabelText("Email address"), request.emailAddress);
    await person.type(screen.getByLabelText("Password"), request.password);

    await person.click(screen.getByRole("button", { name: "Log in" }));
    const button = screen.getByRole("button", { name: "Log in" });
    expect(button).toHaveAttribute("aria-disabled", "true");
    expect(button).toHaveFocus();
    await person.keyboard("{Enter}");
    await person.click(button);

    await screen.findByRole("heading", { level: 1, name: "Admin" });
    expect(requests).toBe(1);
  });

  it("works with the keyboard only", async () => {
    const received: LoginRequest[] = [];
    server.use(...loginHandlers(received));
    renderApp("/admin/login");
    const person = userEvent.setup();

    await person.tab();
    expect(screen.getByLabelText("Email address")).toHaveFocus();
    await person.keyboard("ann@example.com");
    await person.tab();
    expect(screen.getByLabelText("Password")).toHaveFocus();
    await person.keyboard("a password of 12+{Enter}");

    await screen.findByRole("heading", { level: 1, name: "Admin" });
    expect(received).toEqual([{ emailAddress: "ann@example.com", password: "a password of 12+" }]);
  });
});

describe("the Admin area", () => {
  it("sends a person without a session from an Admin page to the login page", async () => {
    server.use(
      http.get<never, never, Problem>(sessionUrl, () =>
        HttpResponse.json(
          problem(401, { detail: "You are not logged in. Log in, then try again." }),
          { status: 401 },
        ),
      ),
    );

    const { router } = renderApp("/admin");

    expect(await screen.findByRole("heading", { level: 1, name: "Log in" })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe("/admin/login");
  });

  it("shows a message while it checks the session, and no Admin content", async () => {
    server.use(
      http.get<never, never, Admin>(sessionUrl, async () => {
        await delay(50);
        return HttpResponse.json(admin());
      }),
    );
    renderApp("/admin");

    expect(screen.getByRole("status")).toHaveTextContent("The application checks your session.");
    expect(screen.queryByRole("button", { name: "Log out" })).not.toBeInTheDocument();
    await screen.findByRole("button", { name: "Log out" });
  });

  it("shows the message of the API and no Admin content when the session check fails with a server error", async () => {
    server.use(
      http.get(sessionUrl, () =>
        HttpResponse.json(problem(500, { detail: serverProblem }), { status: 500 }),
      ),
    );
    renderApp("/admin");

    expect(
      await screen.findByText(serverProblem, undefined, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Log out" })).not.toBeInTheDocument();
  });

  it("logs out, and opens the login page", async () => {
    let loggedOut = false;
    server.use(
      http.get<never, never, Admin | Problem>(sessionUrl, () =>
        loggedOut
          ? HttpResponse.json(problem(401, { detail: "You are not logged in." }), { status: 401 })
          : HttpResponse.json(admin({ emailAddress: "ann@example.com" })),
      ),
      http.delete(sessionUrl, () => {
        loggedOut = true;
        return new HttpResponse(null, { status: 204 });
      }),
    );
    const { router } = renderApp("/admin");
    const person = userEvent.setup();

    await person.click(await screen.findByRole("button", { name: "Log out" }));

    expect(await screen.findByRole("heading", { level: 1, name: "Log in" })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe("/admin/login");
    expect(router.state.historyAction).toBe("REPLACE");
  });

  it("shows the error when the logout fails, and stays on the Admin page", async () => {
    server.use(
      http.get<never, never, Admin>(sessionUrl, () => HttpResponse.json(admin())),
      http.delete(sessionUrl, () =>
        HttpResponse.json(problem(500, { detail: serverProblem }), { status: 500 }),
      ),
    );
    const { router } = renderApp("/admin");
    const person = userEvent.setup();

    await person.click(await screen.findByRole("button", { name: "Log out" }));

    expect(await screen.findByText(serverProblem)).toBeInTheDocument();
    expect(router.state.location.pathname).toBe("/admin");
  });
});

describe("the demo Admin on the login page", () => {
  // test/setup.ts empties both settings, so a test that does not set them sees no demo Admin.
  afterEach(() => {
    vi.stubEnv("VITE_DEMO_ADMIN_EMAIL", "");
    vi.stubEnv("VITE_DEMO_ADMIN_PASSWORD", "");
  });

  describe("when both settings have a value", () => {
    beforeEach(() => {
      vi.stubEnv("VITE_DEMO_ADMIN_EMAIL", "admin@example.com");
      vi.stubEnv("VITE_DEMO_ADMIN_PASSWORD", "demo-admin-password");
    });

    it("shows the email address and the password of the demo Admin", () => {
      renderApp("/admin/login");

      const demo = screen.getByRole("region", { name: "Demo Admin" });
      expect(demo).toHaveTextContent("admin@example.com");
      expect(demo).toHaveTextContent("demo-admin-password");
    });

    it("does not fill the form by itself, so the person enters the values", () => {
      renderApp("/admin/login");

      expect(screen.getByLabelText("Email address")).toHaveValue("");
      expect(screen.getByLabelText("Password")).toHaveValue("");
    });
  });

  it.each([
    ["no setting has a value", "", ""],
    ["only the email address has a value", "admin@example.com", ""],
    ["only the password has a value", "", "demo-admin-password"],
  ])("shows no demo Admin when %s", (_name, email, password) => {
    vi.stubEnv("VITE_DEMO_ADMIN_EMAIL", email);
    vi.stubEnv("VITE_DEMO_ADMIN_PASSWORD", password);
    renderApp("/admin/login");

    expect(screen.queryByRole("region", { name: "Demo Admin" })).not.toBeInTheDocument();
    expect(screen.queryByText("demo-admin-password")).not.toBeInTheDocument();
  });
});
