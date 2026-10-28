import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterAll, afterEach, vi } from "vite-plus/test";

import { server } from "./mocks/server";
import { unhandledRequests } from "./mocks/unhandled";

// The .env file of a developer can set the demo Admin. A test must not depend on it, so a test starts with no demo Admin.
vi.stubEnv("VITE_DEMO_ADMIN_EMAIL", "");
vi.stubEnv("VITE_DEMO_ADMIN_PASSWORD", "");

// Start the mock server now, not in beforeAll. The API client keeps the fetch function that exists
// when its module loads, so the mock server must replace fetch before a test imports the client.
// A request without a handler fails, and it also fails the test (see afterEach), also when the app catches the error.
// With a function, MSW sends the request to the network unless the function throws.
server.listen({
  async onUnhandledFrame({ frame, defaults }) {
    unhandledRequests.push(await frame.getUnhandledMessage());
    defaults.error();
    throw new Error("The request has no handler. Add a handler with server.use.");
  },
});

afterEach(() => {
  cleanup();
  server.resetHandlers();
  const sent = unhandledRequests.splice(0);
  if (sent.length > 0) {
    throw new Error(
      `The test sent requests without a handler. Add a handler with server.use.\n${sent.join("\n")}`,
    );
  }
});
afterAll(() => server.close());
