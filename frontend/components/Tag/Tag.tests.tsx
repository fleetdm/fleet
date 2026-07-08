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

  it("disables the clickable button when disabled is set", () => {
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
});
