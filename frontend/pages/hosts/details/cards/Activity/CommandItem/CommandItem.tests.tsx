import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { createMockCommand } from "__mocks__/commandMock";

import CommandItem from "./CommandItem";

describe("CommandItem", () => {
  const mockOnShowDetails = jest.fn();
  const mockOnCancel = jest.fn();

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("renders a cancel button when onCancel is provided", () => {
    const command = createMockCommand({ request_type: "DeviceLock" });

    render(
      <CommandItem
        command={command}
        onShowDetails={mockOnShowDetails}
        onCancel={mockOnCancel}
      />
    );

    expect(
      screen.getByRole("button", { name: "cancel action" })
    ).toBeInTheDocument();
  });

  it("does not render a cancel button without onCancel", () => {
    const command = createMockCommand({ request_type: "DeviceLock" });

    render(<CommandItem command={command} onShowDetails={mockOnShowDetails} />);

    expect(
      screen.queryByRole("button", { name: "cancel action" })
    ).not.toBeInTheDocument();
  });

  it("renders a cancel button on a deferred (NotNow) command", () => {
    const command = createMockCommand({
      request_type: "DeviceLock",
      status: "NotNow",
    });

    render(
      <CommandItem
        command={command}
        onShowDetails={mockOnShowDetails}
        onCancel={mockOnCancel}
      />
    );

    expect(screen.getByText(/is deferred/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "cancel action" })
    ).toBeInTheDocument();
  });

  it("clicking cancel calls onCancel with the command and does not open details", async () => {
    const command = createMockCommand({ request_type: "DeviceLock" });

    render(
      <CommandItem
        command={command}
        onShowDetails={mockOnShowDetails}
        onCancel={mockOnCancel}
      />
    );

    await userEvent.click(
      screen.getByRole("button", { name: "cancel action" })
    );

    expect(mockOnCancel).toHaveBeenCalledWith(command);
    expect(mockOnShowDetails).not.toHaveBeenCalled();
  });
});
