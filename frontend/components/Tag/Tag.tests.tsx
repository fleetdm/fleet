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

  it.each([
    {
      case: "clickable",
      renderTag: () =>
        render(
          <Tag type="clickable" onClick={() => undefined} disabled>
            iPadOS
          </Tag>
        ),
      buttonName: "iPadOS",
    },
    {
      case: "dismissible",
      renderTag: () =>
        render(
          <Tag type="dismissible" onDismiss={() => undefined} disabled>
            Apple Silicon macOS hosts
          </Tag>
        ),
      buttonName: "Dismiss",
    },
  ])(
    "disables the $case tag's button when disabled is set",
    ({ renderTag, buttonName }) => {
      renderTag();

      expect(screen.getByRole("button", { name: buttonName })).toBeDisabled();
    }
  );

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
});
