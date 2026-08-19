import React from "react";
import { render, screen } from "@testing-library/react";

import DataSet from "./DataSet";

describe("DataSet", () => {
  it("renders title and value", () => {
    render(<DataSet title="Author" value="Alice" />);
    expect(screen.getByText("Author")).toBeInTheDocument();
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("uses vertical orientation by default (no horizontal class, no colon)", () => {
    const { container } = render(<DataSet title="Author" value="Alice" />);
    expect(container.querySelector(".data-set__horizontal")).toBeNull();
    expect(container.querySelector("dt")?.textContent).toBe("Author");
  });

  it("applies horizontal orientation class and appends a colon to the title", () => {
    const { container } = render(
      <DataSet title="Author" value="Alice" orientation="horizontal" />
    );
    expect(
      container.querySelector(".data-set__horizontal")
    ).toBeInTheDocument();
    expect(container.querySelector("dt")?.textContent).toBe("Author:");
  });

  it("applies the text-only modifier when textOnly is set", () => {
    const { container } = render(
      <DataSet title="Author" value="Alice" textOnly />
    );
    expect(container.querySelector(".data-set--text-only")).toBeInTheDocument();
  });

  it("applies the multiline modifier when multiline is set", () => {
    const { container } = render(
      <DataSet title="Resolve" value="Long remediation text." multiline />
    );
    expect(container.querySelector(".data-set--multiline")).toBeInTheDocument();
  });

  it("does not apply the multiline modifier by default", () => {
    const { container } = render(<DataSet title="Author" value="Alice" />);
    expect(container.querySelector(".data-set--multiline")).toBeNull();
  });

  it("merges a custom className alongside the base class", () => {
    const { container } = render(
      <DataSet title="Author" value="Alice" className="custom-class" />
    );
    const root = container.firstChild as HTMLElement;
    expect(root).toHaveClass("data-set");
    expect(root).toHaveClass("custom-class");
  });
});
