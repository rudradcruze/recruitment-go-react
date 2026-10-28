import { screen } from "@testing-library/react";
import { describe, expect, it } from "vite-plus/test";

import { renderApp } from "../../render";

describe("the home page", () => {
  it("shows the name of the application", () => {
    renderApp("/");

    expect(
      screen.getByRole("heading", { level: 1, name: "go-react-template" }),
    ).toBeInTheDocument();
  });

  it("has a page title", () => {
    renderApp("/");

    expect(document.title).toBe("Home - go-react-template");
  });
});
