import { describe, expect, it } from "vite-plus/test";

import { createQueryClient } from "../../../src/api/query-client";
import { shouldRetry } from "../../../src/api/errors";

describe("createQueryClient", () => {
  it("makes a client that retries a query by the rule of shouldRetry", () => {
    expect(createQueryClient().getDefaultOptions().queries?.retry).toBe(shouldRetry);
  });

  it("makes a client that keeps a query result fresh for 30 seconds", () => {
    expect(createQueryClient().getDefaultOptions().queries?.staleTime).toBe(30_000);
  });

  it("makes a client that does not retry a mutation, so a failed request is not sent again", () => {
    expect(createQueryClient().getDefaultOptions().mutations?.retry ?? 0).toBe(0);
  });
});
