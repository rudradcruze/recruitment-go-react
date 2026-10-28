import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vite-plus/test";

import { api } from "../../../src/api/client";
import { server } from "../../mocks/server";

describe("the API client", () => {
  it("sends requests to /api/v1 of the page origin, with the types of the spec", async () => {
    server.use(http.post("/api/v1/admin/session", () => new HttpResponse(null, { status: 204 })));

    const { response } = await api.POST("/admin/session", {
      body: { emailAddress: "admin@example.com", password: "a password of 12+" },
    });

    expect(response.status).toBe(204);
  });
});
