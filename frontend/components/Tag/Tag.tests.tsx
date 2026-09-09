import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import Tag from "./Tag";

describe("Tag", () => {
  it("renders static tags as non-interactive text", () => {
    render(<Tag>Inherited</Tag>);

    expect(screen.getByText("Inherited")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("defaults to the large size", () => {
    render(<Tag>Inherited</Tag>);

    expect(screen.getByText("Inherited")).not.toHaveClass("tag--small");
  });

  it("adds the small modifier class when size is set to small", () => {
    render(<Tag size="small">Inherited</Tag>);

    expect(screen.getByText("Inherited")).toHaveClass("tag--small");
  });

  it("adds the xsmall modifier class when size is set to xsmall", () => {
    render(<Tag size="xsmall">Inherited</Tag>);

    expect(screen.getByText("Inherited")).toHaveClass("tag--xsmall");
  });

  it("does not wrap the tag in a tooltip when tooltip is omitted", () => {
    const { container } = render(<Tag>Inherited</Tag>);

    expect(container.querySelector(".component__tooltip-wrapper")).toBeNull();
  });

  it("wraps the tag in a tooltip when tooltip is provided", () => {
    const { container } = render(
      <Tag tooltip="This report runs on all hosts.">Inherited</Tag>
    );

    expect(screen.getByText("Inherited")).toBeInTheDocument();
    expect(
      container.querySelector(".component__tooltip-wrapper")
    ).not.toBeNull();
  });

  it("renders clickable tags as a button and calls onClick", async () => {
    const handler = jest.fn();
    render(
      <Tag type="clickable" onClick={handler}>
        iPadOS
      </Tag>
    );

    const button = screen.getByRole("button", { name: "iPadOS" });
    await userEvent.click(button);
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("disables the clickable tag's button when disabled is set", () => {
    render(
      <Tag type="clickable" onClick={() => undefined} disabled>
        iPadOS
      </Tag>
    );

    expect(screen.getByRole("button", { name: "iPadOS" })).toBeDisabled();
  });

  it("renders dismissible tags with a dismiss button and calls onDismiss", async () => {
    const handler = jest.fn();
    render(
      <Tag type="dismissible" onDismiss={handler}>
        Apple Silicon macOS hosts
      </Tag>
    );

    expect(screen.getByText("Apple Silicon macOS hosts")).toBeInTheDocument();
    const dismissButton = screen.getByRole("button");
    await userEvent.click(dismissButton);
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("gives the dismiss button an accessible name even when dismissLabel is omitted", () => {
    render(
      <Tag type="dismissible" onDismiss={() => undefined}>
        Apple Silicon macOS hosts
      </Tag>
    );

    expect(screen.getByRole("button", { name: "Dismiss" })).toBeInTheDocument();
  });

  it("uses dismissLabel as the dismiss button's accessible name when provided", () => {
    render(
      <Tag
        type="dismissible"
        onDismiss={() => undefined}
        dismissLabel="Apple Silicon macOS hosts"
      >
        Apple Silicon macOS hosts
      </Tag>
    );

    expect(
      screen.getByRole("button", { name: "Apple Silicon macOS hosts" })
    ).toBeInTheDocument();
  });

  it("does not render a native title tooltip on the dismiss button (aria-label carries the accessible name)", () => {
    render(
      <Tag type="dismissible" onDismiss={() => undefined}>
        Apple Silicon macOS hosts
      </Tag>
    );

    expect(screen.getByRole("button", { name: "Dismiss" })).not.toHaveAttribute(
      "title"
    );
  });

  it.each([
    {
      case: "static",
      renderTag: () => render(<Tag className="custom-tag">Inherited</Tag>),
      label: "Inherited",
    },
    {
      case: "clickable",
      renderTag: () =>
        render(
          <Tag
            type="clickable"
            className="custom-tag"
            onClick={() => undefined}
          >
            iPadOS
          </Tag>
        ),
      label: "iPadOS",
    },
    {
      case: "dismissible",
      renderTag: () =>
        render(
          <Tag
            type="dismissible"
            className="custom-tag"
            onDismiss={() => undefined}
          >
            Apple Silicon macOS hosts
          </Tag>
        ),
      label: "Apple Silicon macOS hosts",
    },
  ])("applies className to the root of a $case tag", ({ renderTag, label }) => {
    renderTag();

    expect(screen.getByText(label).closest(".tag")).toHaveClass("custom-tag");
  });
});
