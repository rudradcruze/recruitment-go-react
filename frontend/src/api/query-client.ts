import { QueryClient, type QueryOptions } from "@tanstack/react-query";

import { shouldRetry } from "./errors";

// The time that a query result is fresh. A fresh result does not make a request on a new page or on a window focus.
const freshTimeMs = 30_000;

// createQueryClient makes the query client of the application. A failed mutation is not sent again,
// because the person did not ask for a second request. A test can give another retry rule.
export function createQueryClient(retry: QueryOptions["retry"] = shouldRetry): QueryClient {
  return new QueryClient({ defaultOptions: { queries: { retry, staleTime: freshTimeMs } } });
}
