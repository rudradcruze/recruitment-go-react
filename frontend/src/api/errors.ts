const serverMessage = "The server cannot complete the request. Try again in 1 minute.";
const notFoundMessage = "The server cannot find the data. Check the URL of the page.";
const rejectedMessage = "The server did not accept the request. Check the values. Then try again.";

// ApiError is the error of a request that the API did not accept. The message is a text for the person.
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

// detailOf gives the detail of a problem detail (RFC 9457), or undefined when the body has no text detail.
// The body can be anything: a proxy can send an empty body or an HTML page.
function detailOf(body: unknown): string | undefined {
  if (typeof body === "object" && body !== null && "detail" in body) {
    return typeof body.detail === "string" && body.detail !== "" ? body.detail : undefined;
  }
  return undefined;
}

// fallbackMessage is the text for a failed response that has no text detail.
function fallbackMessage(status: number): string {
  if (status >= 500) {
    return serverMessage;
  }
  return status === 404 ? notFoundMessage : rejectedMessage;
}

// unwrap gives the data of a response that succeeded. Otherwise it throws an ApiError.
// It checks the status of the response, because a failed response can have no body.
export function unwrap<T>({
  data,
  error,
  response,
}: {
  data?: T;
  error?: unknown;
  response: Response;
}): T {
  if (response.ok && data !== undefined) {
    return data;
  }
  throw new ApiError(response.status, detailOf(error) ?? fallbackMessage(response.status));
}

// unwrapEmpty is like unwrap, for a response that has no body when it succeeds, for example 204.
export function unwrapEmpty({ error, response }: { error?: unknown; response: Response }): void {
  if (!response.ok) {
    throw new ApiError(response.status, detailOf(error) ?? fallbackMessage(response.status));
  }
}

const maxRetries = 2;

// shouldRetry tells TanStack Query whether it must send a failed query again.
// A response with a 4xx status does not change when the client sends it again.
// A 5xx response and a network error can be temporary.
export function shouldRetry(failureCount: number, error: Error): boolean {
  if (error instanceof ApiError && error.status < 500) {
    return false;
  }
  return failureCount < maxRetries;
}

// messageOf gives the text of an error for the person.
// For an ApiError, it is the message of the error. A network error has no response, so it has its own text.
export function messageOf(error: Error): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  return "The application cannot reach the server. Check your connection. Then try again.";
}
