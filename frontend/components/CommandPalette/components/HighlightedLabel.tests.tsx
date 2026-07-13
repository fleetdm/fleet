import React from "react";
import { render, screen } from "@testing-library/react";

import HighlightedLabel from "./HighlightedLabel";

describe("HighlightedLabel", () => {
  it("renders the text plainly when query is empty", () => {
    const { container } = render(
      <HighlightedLabel text="Engineering" query="" />
    );
    expect(screen.getByText("Engineering")).toBeInTheDocument();
    expect(container.querySelectorAll("mark")).toHaveLength(0);
  });

  it("wraps the matched substring in a <mark>", () => {
    const { container } = render(
      <HighlightedLabel text="Engineering" query="gin" />
    );
    const marks = container.querySelectorAll("mark");
    expect(marks).toHaveLength(1);
    expect(marks[0].textContent).toBe("gin");
    expect(marks[0]).toHaveClass("command-palette__item-label-match");
  });

  it("matches case-insensitively while preserving original case", () => {
    const { container } = render(
      <HighlightedLabel text="Engineering" query="ENG" />
    );
    const marks = container.querySelectorAll("mark");
    expect(marks).toHaveLength(1);
    expect(marks[0].textContent).toBe("Eng");
  });

  it("highlights each token of a multi-token query", () => {
    const { container } = render(
      <HighlightedLabel text="View host details" query="view det" />
    );
    const marks = container.querySelectorAll("mark");
    expect(marks).toHaveLength(2);
    expect(marks[0].textContent).toBe("View");
    expect(marks[1].textContent).toBe("det");
  });

  it("renders plain text when the query has no match", () => {
    const { container } = render(
      <HighlightedLabel text="Engineering" query="zzz" />
    );
    expect(screen.getByText("Engineering")).toBeInTheDocument();
    expect(container.querySelectorAll("mark")).toHaveLength(0);
  });
});
