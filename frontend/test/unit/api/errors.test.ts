import { describe, expect, it } from "vite-plus/test";

import { ApiError, messageOf, shouldRetry, unwrap, unwrapEmpty } from "../../../src/api/errors";

describe("unwrap", () => {
  it("gives the data of a response that succeeded", () => {
    expect(unwrap({ data: { id: 1 }, response: new Response(null, { status: 201 }) })).toEqual({
      id: 1,
    });
  });

  it("throws an ApiError with the status and the text detail of the problem", () => {
    const failure = () =>
      unwrap({
        error: { detail: "The email address is already registered." },
        response: new Response(null, { status: 409 }),
      });

    expect(failure).toThrow(new ApiError(409, "The email address is already registered."));
  });

  it("throws an ApiError for a success status that has no data", () => {
    expect(() => unwrap({ response: new Response(null, { status: 200 }) })).toThrow(ApiError);
  });
});

describe("unwrapEmpty", () => {
  it("gives nothing for a response that succeeded without a body", () => {
    expect(unwrapEmpty({ response: new Response(null, { status: 204 }) })).toBeUndefined();
  });

  it("throws an ApiError with the status and the text detail of a failed response", () => {
    const failure = () =>
      unwrapEmpty({
        error: { detail: "The Attempt is closed." },
        response: new Response(null, { status: 409 }),
      });

    expect(failure).toThrow(new ApiError(409, "The Attempt is closed."));
  });

  it("throws an ApiError with a general message for a failed response without a body", () => {
    expect(() => unwrapEmpty({ response: new Response(null, { status: 502 }) })).toThrow(
      new ApiError(502, "The server cannot complete the request. Try again in 1 minute."),
    );
  });
});

describe("unwrap, the message of a failed response without a text detail", () => {
  const messageFor = (status: number) => {
    try {
      unwrap({ error: "", response: new Response(null, { status }) });
    } catch (error) {
      return (error as ApiError).message;
    }
  };

  it("tells to try again for a 5xx status", () => {
    expect(messageFor(502)).toBe("The server cannot complete the request. Try again in 1 minute.");
  });

  it("tells to check the URL for a 404 status", () => {
    expect(messageFor(404)).toBe("The server cannot find the data. Check the URL of the page.");
  });

  it("tells to check the values for another 4xx status", () => {
    expect(messageFor(422)).toBe(
      "The server did not accept the request. Check the values. Then try again.",
    );
  });
});

describe("messageOf", () => {
  it("gives the message of an ApiError", () => {
    expect(messageOf(new ApiError(409, "The email address is already registered."))).toBe(
      "The email address is already registered.",
    );
  });

  it("gives a text about the connection for an error without a response", () => {
    expect(messageOf(new TypeError("Failed to fetch"))).toContain(
      "The application cannot reach the server.",
    );
  });
});

describe("shouldRetry", () => {
  const notFound = new ApiError(404, "The User does not exist.");
  const unavailable = new ApiError(503, "The server cannot complete the request.");
  const networkError = new TypeError("Failed to fetch");

  it("does not retry a response that the server rejected with a 4xx status", () => {
    expect(shouldRetry(0, notFound)).toBe(false);
    expect(shouldRetry(0, new ApiError(422, "The email address is not valid."))).toBe(false);
  });

  it("retries a 5xx response", () => {
    expect(shouldRetry(0, unavailable)).toBe(true);
    expect(shouldRetry(1, unavailable)).toBe(true);
  });

  it("retries a network error", () => {
    expect(shouldRetry(0, networkError)).toBe(true);
  });

  it("stops after 2 retries", () => {
    expect(shouldRetry(2, unavailable)).toBe(false);
    expect(shouldRetry(2, networkError)).toBe(false);
  });
});
